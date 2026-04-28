package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var trustedCIDRs []*net.IPNet

func initTrustedProxies() {
	defaults := []string{
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"127.0.0.0/8", "::1/128", "fc00::/7",
	}
	if extra := os.Getenv("TRUSTED_PROXIES"); extra != "" {
		defaults = append(defaults, strings.Split(extra, ",")...)
	}
	for _, cidr := range defaults {
		_, network, err := net.ParseCIDR(strings.TrimSpace(cidr))
		if err == nil {
			trustedCIDRs = append(trustedCIDRs, network)
		}
	}
}

func isTrustedProxy(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, cidr := range trustedCIDRs {
		if cidr.Contains(parsed) {
			return true
		}
	}
	return false
}

func getClientIP(c *gin.Context) string {
	remoteIP, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		remoteIP = c.Request.RemoteAddr
	}
	if !isTrustedProxy(remoteIP) {
		return remoteIP
	}
	if cfIP := c.GetHeader("CF-Connecting-IP"); cfIP != "" {
		ip := strings.TrimSpace(cfIP)
		if parsed := net.ParseIP(ip); parsed != nil {
			return ip
		}
	}
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if parsed := net.ParseIP(ip); parsed != nil {
			return ip
		}
	}
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		ip := strings.TrimSpace(xri)
		if parsed := net.ParseIP(ip); parsed != nil {
			return ip
		}
	}
	return remoteIP
}

func adminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := os.Getenv("ADMIN_TOKEN")
		if token == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ADMIN_TOKEN not configured"})
			c.Abort()
			return
		}
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != token {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func handleIssue(c *gin.Context) {
	ip := getClientIP(c)
	fmt.Printf("[issue] client_ip=%s remote_addr=%s x_forwarded_for=%s x_real_ip=%s\n",
		ip, c.Request.RemoteAddr, c.GetHeader("X-Forwarded-For"), c.GetHeader("X-Real-IP"))
	license, err := dbFindLicenseByIP(ip)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	if license == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "该IP未被授权", "detected_ip": ip})
		return
	}
	c.Header("X-Detected-IP", ip)
	c.JSON(http.StatusOK, license)
}

type createReq struct {
	AllowedIPs []string `json:"allowed_ips" binding:"required"`
	ExpiresAt  *string  `json:"expires_at"`
}

func handleCreate(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}
	licenseID := uuid.New().String()
	issuedAt := time.Now().UTC().Format(time.RFC3339)
	if err := dbCreateLicense(licenseID, req.AllowedIPs, issuedAt, req.ExpiresAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed: " + err.Error()})
		return
	}
	l := &License{
		LicenseID:  licenseID,
		AllowedIPs: req.AllowedIPs,
		IssuedAt:   issuedAt,
		ExpiresAt:  req.ExpiresAt,
	}
	signLicense(l)
	c.JSON(http.StatusOK, l)
}

func handleList(c *gin.Context) {
	licenses, err := dbListLicenses()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed: " + err.Error()})
		return
	}
	if licenses == nil {
		licenses = []License{}
	}
	c.JSON(http.StatusOK, licenses)
}

type revokeReq struct {
	LicenseID string `json:"license_id" binding:"required"`
}

func handleRevoke(c *gin.Context) {
	var req revokeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := dbDeleteLicense(req.LicenseID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "revoked"})
}

type updateReq struct {
	LicenseID  string   `json:"license_id" binding:"required"`
	AllowedIPs []string `json:"allowed_ips" binding:"required"`
	ExpiresAt  *string  `json:"expires_at"`
}

func handleUpdate(c *gin.Context) {
	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := dbUpdateLicense(req.LicenseID, req.AllowedIPs, req.ExpiresAt); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}
