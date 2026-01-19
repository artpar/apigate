package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/domain/tls"
	"github.com/spf13/cobra"
)

var certificatesCmd = &cobra.Command{
	Use:   "certificates",
	Short: "Manage TLS certificates",
	Long: `Manage TLS certificates stored in the database.

Certificates are used for HTTPS/TLS. You can:
- List all certificates
- View certificate details
- Upload manual certificates
- Check expiring/expired certificates
- Revoke or delete certificates

Note: ACME certificates are automatically managed when TLS mode is 'acme'.

Examples:
  apigate certificates list
  apigate certificates get <id>
  apigate certificates get-domain api.example.com
  apigate certificates expiring --days 30
  apigate certificates expired`,
}

var certificatesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all certificates",
	RunE:  runCertificatesList,
}

var certificatesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a certificate by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runCertificatesGet,
}

var certificatesGetDomainCmd = &cobra.Command{
	Use:   "get-domain <domain>",
	Short: "Get a certificate by domain",
	Args:  cobra.ExactArgs(1),
	RunE:  runCertificatesGetDomain,
}

var certificatesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create/upload a certificate (manual mode)",
	Long: `Upload a manual certificate to the database.

This is used for manual TLS mode where you provide your own certificates.
For ACME mode, certificates are obtained automatically.

Examples:
  apigate certificates create \
    --domain api.example.com \
    --cert-pem /path/to/cert.pem \
    --key-pem /path/to/key.pem \
    --chain-pem /path/to/chain.pem \
    --expires-at "2026-01-19T00:00:00Z"`,
	RunE: runCertificatesCreate,
}

var certificatesExpiringCmd = &cobra.Command{
	Use:   "expiring",
	Short: "List certificates expiring within N days",
	RunE:  runCertificatesExpiring,
}

var certificatesExpiredCmd = &cobra.Command{
	Use:   "expired",
	Short: "List expired certificates",
	RunE:  runCertificatesExpired,
}

var certificatesRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke a certificate",
	Args:  cobra.ExactArgs(1),
	RunE:  runCertificatesRevoke,
}

var certificatesDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a certificate",
	Args:  cobra.ExactArgs(1),
	RunE:  runCertificatesDelete,
}

// Flags for create command
var (
	certDomain    string
	certCertPEM   string
	certKeyPEM    string
	certChainPEM  string
	certExpiresAt string
)

// Flags for expiring command
var certExpiringDays int

// Flags for revoke command
var certRevokeReason string

func init() {
	rootCmd.AddCommand(certificatesCmd)

	certificatesCmd.AddCommand(certificatesListCmd)
	certificatesCmd.AddCommand(certificatesGetCmd)
	certificatesCmd.AddCommand(certificatesGetDomainCmd)
	certificatesCmd.AddCommand(certificatesCreateCmd)
	certificatesCmd.AddCommand(certificatesExpiringCmd)
	certificatesCmd.AddCommand(certificatesExpiredCmd)
	certificatesCmd.AddCommand(certificatesRevokeCmd)
	certificatesCmd.AddCommand(certificatesDeleteCmd)

	// Create command flags
	certificatesCreateCmd.Flags().StringVar(&certDomain, "domain", "", "domain for the certificate (required)")
	certificatesCreateCmd.Flags().StringVar(&certCertPEM, "cert-pem", "", "path to certificate PEM file (required)")
	certificatesCreateCmd.Flags().StringVar(&certKeyPEM, "key-pem", "", "path to private key PEM file (required)")
	certificatesCreateCmd.Flags().StringVar(&certChainPEM, "chain-pem", "", "path to certificate chain PEM file")
	certificatesCreateCmd.Flags().StringVar(&certExpiresAt, "expires-at", "", "expiration time (RFC3339 format)")
	_ = certificatesCreateCmd.MarkFlagRequired("domain")
	_ = certificatesCreateCmd.MarkFlagRequired("cert-pem")
	_ = certificatesCreateCmd.MarkFlagRequired("key-pem")

	// Expiring command flags
	certificatesExpiringCmd.Flags().IntVar(&certExpiringDays, "days", 30, "number of days to check")

	// Revoke command flags
	certificatesRevokeCmd.Flags().StringVar(&certRevokeReason, "reason", "", "reason for revocation")
}

func runCertificatesList(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	certStore := sqlite.NewCertificateStore(db)
	certs, err := certStore.List(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list certificates: %w", err)
	}

	if len(certs) == 0 {
		fmt.Println("No certificates found.")
		fmt.Println("\nTo enable ACME (Let's Encrypt), configure TLS settings:")
		fmt.Println("  apigate settings set tls.enabled true")
		fmt.Println("  apigate settings set tls.mode acme")
		fmt.Println("  apigate settings set tls.domain your-domain.com")
		fmt.Println("  apigate settings set tls.acme_email admin@your-domain.com")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDOMAIN\tSTATUS\tISSUER\tEXPIRES\tDAYS LEFT")
	fmt.Fprintln(w, "---\t------\t------\t------\t-------\t---------")

	for _, cert := range certs {
		daysLeft := cert.DaysUntilExpiry()
		daysLeftStr := fmt.Sprintf("%d", daysLeft)
		if daysLeft < 0 {
			daysLeftStr = "EXPIRED"
		} else if daysLeft <= 7 {
			daysLeftStr = fmt.Sprintf("%d (critical)", daysLeft)
		} else if daysLeft <= 30 {
			daysLeftStr = fmt.Sprintf("%d (warning)", daysLeft)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			cert.ID,
			cert.Domain,
			cert.Status,
			truncateString(cert.Issuer, 20),
			cert.ExpiresAt.Format("2006-01-02"),
			daysLeftStr,
		)
	}

	w.Flush()
	return nil
}

func runCertificatesGet(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	certStore := sqlite.NewCertificateStore(db)
	cert, err := certStore.Get(context.Background(), args[0])
	if err != nil {
		return fmt.Errorf("certificate not found: %s", args[0])
	}

	printCertificateDetails(cert)
	return nil
}

func runCertificatesGetDomain(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	certStore := sqlite.NewCertificateStore(db)
	cert, err := certStore.GetByDomain(context.Background(), args[0])
	if err != nil {
		return fmt.Errorf("certificate not found for domain: %s", args[0])
	}

	printCertificateDetails(cert)
	return nil
}

func runCertificatesCreate(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	// Read certificate files
	certPEM, err := os.ReadFile(certCertPEM)
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %w", err)
	}

	keyPEM, err := os.ReadFile(certKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	var chainPEM []byte
	if certChainPEM != "" {
		chainPEM, err = os.ReadFile(certChainPEM)
		if err != nil {
			return fmt.Errorf("failed to read chain file: %w", err)
		}
	}

	// Parse expiration time
	var expiresAt time.Time
	if certExpiresAt != "" {
		expiresAt, err = time.Parse(time.RFC3339, certExpiresAt)
		if err != nil {
			return fmt.Errorf("invalid expires-at format (use RFC3339): %w", err)
		}
	} else {
		// Default to 90 days from now
		expiresAt = time.Now().UTC().AddDate(0, 0, 90)
	}

	now := time.Now().UTC()
	cert := tls.Certificate{
		ID:        tls.GenerateCertificateID(),
		Domain:    certDomain,
		CertPEM:   certPEM,
		KeyPEM:    keyPEM,
		ChainPEM:  chainPEM,
		IssuedAt:  now,
		ExpiresAt: expiresAt,
		Issuer:    "Manual",
		Status:    tls.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	certStore := sqlite.NewCertificateStore(db)
	if err := certStore.Create(context.Background(), cert); err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	fmt.Printf("%s Created certificate: %s for domain %s\n", checkMark, cert.ID, cert.Domain)
	fmt.Printf("   Expires: %s (%d days)\n", cert.ExpiresAt.Format("2006-01-02"), cert.DaysUntilExpiry())
	return nil
}

func runCertificatesExpiring(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	certStore := sqlite.NewCertificateStore(db)
	certs, err := certStore.ListExpiring(context.Background(), certExpiringDays)
	if err != nil {
		return fmt.Errorf("failed to list expiring certificates: %w", err)
	}

	if len(certs) == 0 {
		fmt.Printf("No certificates expiring within %d days.\n", certExpiringDays)
		return nil
	}

	fmt.Printf("Certificates expiring within %d days:\n\n", certExpiringDays)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDOMAIN\tEXPIRES\tDAYS LEFT")
	fmt.Fprintln(w, "---\t------\t-------\t---------")

	for _, cert := range certs {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n",
			cert.ID,
			cert.Domain,
			cert.ExpiresAt.Format("2006-01-02"),
			cert.DaysUntilExpiry(),
		)
	}

	w.Flush()
	return nil
}

func runCertificatesExpired(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	certStore := sqlite.NewCertificateStore(db)
	certs, err := certStore.ListExpired(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list expired certificates: %w", err)
	}

	if len(certs) == 0 {
		fmt.Println("No expired certificates.")
		return nil
	}

	fmt.Println("Expired certificates:")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDOMAIN\tEXPIRED ON\tDAYS AGO")
	fmt.Fprintln(w, "---\t------\t----------\t---------")

	for _, cert := range certs {
		daysAgo := -cert.DaysUntilExpiry()
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n",
			cert.ID,
			cert.Domain,
			cert.ExpiresAt.Format("2006-01-02"),
			daysAgo,
		)
	}

	w.Flush()
	return nil
}

func runCertificatesRevoke(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	certStore := sqlite.NewCertificateStore(db)
	cert, err := certStore.Get(context.Background(), args[0])
	if err != nil {
		return fmt.Errorf("certificate not found: %s", args[0])
	}

	now := time.Now().UTC()
	cert.Status = tls.StatusRevoked
	cert.RevokedAt = &now
	cert.RevokeReason = certRevokeReason
	cert.UpdatedAt = now

	if err := certStore.Update(context.Background(), cert); err != nil {
		return fmt.Errorf("failed to revoke certificate: %w", err)
	}

	fmt.Printf("%s Revoked certificate: %s (%s)\n", checkMark, cert.ID, cert.Domain)
	if certRevokeReason != "" {
		fmt.Printf("   Reason: %s\n", certRevokeReason)
	}
	return nil
}

func runCertificatesDelete(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	certStore := sqlite.NewCertificateStore(db)
	if err := certStore.Delete(context.Background(), args[0]); err != nil {
		return fmt.Errorf("failed to delete certificate: %w", err)
	}

	fmt.Printf("%s Deleted certificate: %s\n", checkMark, args[0])
	return nil
}

func printCertificateDetails(cert tls.Certificate) {
	fmt.Println("Certificate Details:")
	fmt.Println("--------------------")
	fmt.Printf("  ID:            %s\n", cert.ID)
	fmt.Printf("  Domain:        %s\n", cert.Domain)
	fmt.Printf("  Status:        %s\n", cert.Status)
	fmt.Printf("  Issuer:        %s\n", cert.Issuer)
	fmt.Printf("  Serial Number: %s\n", cert.SerialNumber)
	fmt.Printf("  Issued At:     %s\n", cert.IssuedAt.Format(time.RFC3339))
	fmt.Printf("  Expires At:    %s\n", cert.ExpiresAt.Format(time.RFC3339))
	fmt.Printf("  Days Left:     %d\n", cert.DaysUntilExpiry())

	if cert.ACMEAccountURL != "" {
		fmt.Printf("  ACME Account:  %s\n", cert.ACMEAccountURL)
	}

	if cert.RevokedAt != nil {
		fmt.Printf("  Revoked At:    %s\n", cert.RevokedAt.Format(time.RFC3339))
		fmt.Printf("  Revoke Reason: %s\n", cert.RevokeReason)
	}

	fmt.Printf("  Created At:    %s\n", cert.CreatedAt.Format(time.RFC3339))
	fmt.Printf("  Updated At:    %s\n", cert.UpdatedAt.Format(time.RFC3339))
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
