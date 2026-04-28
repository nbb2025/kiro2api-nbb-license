package main

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func initDB(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS licenses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		license_id TEXT UNIQUE NOT NULL,
		allowed_ips TEXT NOT NULL,
		issued_at TEXT NOT NULL,
		expires_at TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	return nil
}

func dbCreateLicense(licenseID string, allowedIPs []string, issuedAt string, expiresAt *string) error {
	ipsJSON, _ := json.Marshal(allowedIPs)
	_, err := db.Exec("INSERT INTO licenses (license_id, allowed_ips, issued_at, expires_at) VALUES (?, ?, ?, ?)",
		licenseID, string(ipsJSON), issuedAt, expiresAt)
	return err
}

func dbListLicenses() ([]License, error) {
	rows, err := db.Query("SELECT license_id, allowed_ips, issued_at, expires_at FROM licenses")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var licenses []License
	for rows.Next() {
		var l License
		var ipsJSON string
		var expiresAt sql.NullString
		if err := rows.Scan(&l.LicenseID, &ipsJSON, &l.IssuedAt, &expiresAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(ipsJSON), &l.AllowedIPs)
		if expiresAt.Valid {
			l.ExpiresAt = &expiresAt.String
		}
		if err := signLicense(&l); err != nil {
			return nil, err
		}
		licenses = append(licenses, l)
	}
	return licenses, nil
}

func dbFindLicenseByIP(ip string) (*License, error) {
	rows, err := db.Query("SELECT license_id, allowed_ips, issued_at, expires_at FROM licenses")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var l License
		var ipsJSON string
		var expiresAt sql.NullString
		if err := rows.Scan(&l.LicenseID, &ipsJSON, &l.IssuedAt, &expiresAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(ipsJSON), &l.AllowedIPs)
		if expiresAt.Valid {
			l.ExpiresAt = &expiresAt.String
		}
		for _, allowedIP := range l.AllowedIPs {
			if allowedIP == ip {
				if err := signLicense(&l); err != nil {
					return nil, err
				}
				return &l, nil
			}
		}
	}
	return nil, nil
}

func dbDeleteLicense(licenseID string) error {
	result, err := db.Exec("DELETE FROM licenses WHERE license_id = ?", licenseID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("license not found")
	}
	return nil
}

func dbUpdateLicense(licenseID string, allowedIPs []string, expiresAt *string) error {
	ipsJSON, _ := json.Marshal(allowedIPs)
	result, err := db.Exec("UPDATE licenses SET allowed_ips = ?, expires_at = ? WHERE license_id = ?",
		string(ipsJSON), expiresAt, licenseID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("license not found")
	}
	return nil
}
