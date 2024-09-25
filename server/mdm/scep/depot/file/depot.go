package file

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// NewFileDepot returns a new cert depot.
func NewFileDepot(path string) (*fileDepot, error) {
	f, err := os.OpenFile(fmt.Sprintf("%s/index.txt", path),
		os.O_RDONLY|os.O_CREATE, 0o666)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return &fileDepot{dirPath: path}, nil
}

type fileDepot struct {
	dirPath  string
	serialMu sync.Mutex
	dbMu     sync.Mutex
}

func (d *fileDepot) CA(pass []byte) ([]*x509.Certificate, *rsa.PrivateKey, error) {
	caPEM, err := d.getFile("ca.pem")
	if err != nil {
		return nil, nil, err
	}
	cert, err := loadCert(caPEM.Data)
	if err != nil {
		return nil, nil, err
	}
	keyPEM, err := d.getFile("ca.key")
	if err != nil {
		return nil, nil, err
	}
	key, err := loadKey(keyPEM.Data, pass)
	if err != nil {
		return nil, nil, err
	}
	return []*x509.Certificate{cert}, key, nil
}

// file permissions
const (
	certPerm   = 0o444
	serialPerm = 0o400
	dbPerm     = 0o600
)

// Put adds a certificate to the depot
func (d *fileDepot) Put(cn string, crt *x509.Certificate) error {
	if crt == nil {
		return errors.New("crt is nil")
	}
	if crt.Raw == nil {
		return errors.New("data is nil")
	}
	data := crt.Raw

	if err := os.MkdirAll(d.dirPath, 0o755); err != nil {
		return err
	}

	serial := crt.SerialNumber

	if crt.Subject.CommonName == "" {
		// this means our cn was replaced by the certificate Signature
		// which is inappropriate for a filename
		cn = fmt.Sprintf("%x", sha256.Sum256(crt.Raw))
	}
	filename := fmt.Sprintf("%s.%s.pem", cn, serial.String())

	filepath := d.path(filename)
	file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, certPerm)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(pemCert(data)); err != nil {
		os.Remove(filepath)
		return err
	}
	if err := d.writeDB(cn, serial, filename, crt); err != nil {
		// TODO : remove certificate in case of writeDB problems
		return err
	}

	return nil
}

func (d *fileDepot) Serial() (*big.Int, error) {
	d.serialMu.Lock()
	defer d.serialMu.Unlock()
	name := d.path("serial")
	s := big.NewInt(2)
	if err := d.check("serial"); err != nil {
		// assuming it doesnt exist, create
		if err := d.writeSerial(s); err != nil {
			return nil, err
		}
		return s, nil
	}
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	r := bufio.NewReader(file)
	data, err := r.ReadString('\r')
	if err != nil && err != io.EOF {
		return nil, err
	}
	data = strings.TrimSuffix(data, "\r")
	data = strings.TrimSuffix(data, "\n")
	serial, ok := s.SetString(data, 16)
	if !ok {
		return nil, errors.New("could not convert " + data + " to serial number")
	}
	if err := d.incrementSerial(serial); err != nil {
		return serial, err
	}
	return serial, nil
}

func makeOpenSSLTime(t time.Time) string {
	y := t.Year() % 100
	validDate := fmt.Sprintf("%02d%02d%02d%02d%02d%02dZ", y, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	return validDate
}

func makeDn(cert *x509.Certificate) string {
	var dn bytes.Buffer

	if len(cert.Subject.Country) > 0 && len(cert.Subject.Country[0]) > 0 {
		dn.WriteString("/C=" + cert.Subject.Country[0])
	}
	if len(cert.Subject.Province) > 0 && len(cert.Subject.Province[0]) > 0 {
		dn.WriteString("/ST=" + cert.Subject.Province[0])
	}
	if len(cert.Subject.Locality) > 0 && len(cert.Subject.Locality[0]) > 0 {
		dn.WriteString("/L=" + cert.Subject.Locality[0])
	}
	if len(cert.Subject.Organization) > 0 && len(cert.Subject.Organization[0]) > 0 {
		dn.WriteString("/O=" + cert.Subject.Organization[0])
	}
	if len(cert.Subject.OrganizationalUnit) > 0 && len(cert.Subject.OrganizationalUnit[0]) > 0 {
		dn.WriteString("/OU=" + cert.Subject.OrganizationalUnit[0])
	}
	if len(cert.Subject.CommonName) > 0 {
		dn.WriteString("/CN=" + cert.Subject.CommonName)
	}
	if len(cert.EmailAddresses) > 0 {
		dn.WriteString("/emailAddress=" + cert.EmailAddresses[0])
	}
	return dn.String()
}

// Determine if the cadb already has a valid certificate with the same name
func (d *fileDepot) HasCN(_ string, allowTime int, cert *x509.Certificate, revokeOldCertificate bool) (bool, error) {

	var addDB bytes.Buffer
	candidates := make(map[string]string)

	dn := makeDn(cert)

	if err := os.MkdirAll(d.dirPath, 0o755); err != nil {
		return false, err
	}

	name := d.path("index.txt")
	file, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Loop over index.txt, determine if a certificate is valid and can be revoked
	// revoke certificate in DB if requested
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, dn) {
			// Removing revoked certificate from candidates, if any
			if strings.HasPrefix(line, "R\t") {
				entries := strings.Split(line, "\t")
				serial := strings.ToUpper(entries[3])
				candidates[serial] = line
				delete(candidates, serial)
				addDB.WriteString(line + "\n")
				// Test & add certificate candidates, if any
			} else if strings.HasPrefix(line, "V\t") {
				issueDate, err := strconv.ParseInt(strings.Replace(strings.Split(line, "\t")[1], "Z", "", 1), 10, 64)
				if err != nil {
					return false, errors.New("Could not get expiry date from ca db")
				}
				minimalRenewDate, err := strconv.ParseInt(strings.Replace(makeOpenSSLTime(time.Now().AddDate(0, 0, allowTime).UTC()), "Z", "", 1), 10, 64)
				if err != nil {
					return false, errors.New("Could not calculate expiry date")
				}
				entries := strings.Split(line, "\t")
				serial := strings.ToUpper(entries[3])

				// all non renewable certificates
				if minimalRenewDate < issueDate && allowTime > 0 {
					candidates[serial] = "no"
				} else {
					candidates[serial] = line
				}
			}
		} else {
			addDB.WriteString(line + "\n")
		}
	}
	file.Close()
	for key, value := range candidates {
		if value == "no" {
			return false, errors.New("DN " + dn + " already exists")
		}
		if revokeOldCertificate {
			fmt.Println("Revoking certificate with serial " + key + " from DB. Recreation of CRL needed.")
			entries := strings.Split(value, "\t")
			addDB.WriteString("R\t" + entries[1] + "\t" + makeOpenSSLTime(time.Now()) + "\t" + strings.ToUpper(entries[3]) + "\t" + entries[4] + "\t" + entries[5] + "\n")
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	if revokeOldCertificate {
		file, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR, dbPerm)
		if err != nil {
			return false, err
		}
		if _, err := file.Write(addDB.Bytes()); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (d *fileDepot) writeDB(cn string, serial *big.Int, filename string, cert *x509.Certificate) error {
	d.dbMu.Lock()
	defer d.dbMu.Unlock()

	var dbEntry bytes.Buffer

	// Revoke old certificate
	if _, err := d.HasCN(cn, 0, cert, true); err != nil {
		return err
	}
	if err := os.MkdirAll(d.dirPath, 0o755); err != nil {
		return err
	}
	name := d.path("index.txt")

	file, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR|os.O_APPEND, dbPerm)
	if err != nil {
		return fmt.Errorf("could not append to "+name+" : %q\n", err.Error())
	}
	defer file.Close()

	// Format of the caDB, see http://pki-tutorial.readthedocs.io/en/latest/cadb.html
	//   STATUSFLAG  EXPIRATIONDATE  REVOCATIONDATE(or emtpy)	SERIAL_IN_HEX   CERTFILENAME_OR_'unknown'   Certificate_DN

	serialHex := fmt.Sprintf("%X", cert.SerialNumber)
	if len(serialHex)%2 == 1 {
		serialHex = fmt.Sprintf("0%s", serialHex)
	}

	validDate := makeOpenSSLTime(cert.NotAfter)

	dn := makeDn(cert)

	// Valid
	dbEntry.WriteString("V\t")
	// Valid till
	dbEntry.WriteString(validDate + "\t")
	// Empty (not revoked)
	dbEntry.WriteString("\t")
	// Serial in Hex
	dbEntry.WriteString(serialHex + "\t")
	// Certificate file name
	dbEntry.WriteString(filename + "\t")
	// Certificate DN
	dbEntry.WriteString(dn)
	dbEntry.WriteString("\n")

	if _, err := file.Write(dbEntry.Bytes()); err != nil {
		return err
	}
	return nil
}

func (d *fileDepot) writeSerial(serial *big.Int) error {
	if err := os.MkdirAll(d.dirPath, 0o755); err != nil {
		return err
	}
	name := d.path("serial")
	os.Remove(name)

	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, serialPerm)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(fmt.Sprintf("%x\n", serial.Bytes())); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// read serial and increment
func (d *fileDepot) incrementSerial(s *big.Int) error {
	serial := s.Add(s, big.NewInt(1))
	if err := d.writeSerial(serial); err != nil {
		return err
	}
	return nil
}

type file struct {
	Info os.FileInfo
	Data []byte
}

func (d *fileDepot) check(path string) error {
	name := d.path(path)
	_, err := os.Stat(name)
	if err != nil {
		return err
	}
	return nil
}

func (d *fileDepot) getFile(path string) (*file, error) {
	if err := d.check(path); err != nil {
		return nil, err
	}
	fi, err := os.Stat(d.path(path))
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadFile(d.path(path))
	return &file{fi, b}, err
}

func (d *fileDepot) path(name string) string {
	return filepath.Join(d.dirPath, name)
}

const (
	rsaPrivateKeyPEMBlockType   = "RSA PRIVATE KEY"
	pkcs8PrivateKeyPEMBlockType = "PRIVATE KEY"
	certificatePEMBlockType     = "CERTIFICATE"
)

// load an encrypted private key from disk
func loadKey(data []byte, password []byte) (*rsa.PrivateKey, error) {
	pemBlock, _ := pem.Decode(data)
	if pemBlock == nil {
		return nil, errors.New("PEM decode failed")
	}
	switch pemBlock.Type {
	case rsaPrivateKeyPEMBlockType:
		if x509.IsEncryptedPEMBlock(pemBlock) {
			b, err := x509.DecryptPEMBlock(pemBlock, password)
			if err != nil {
				return nil, err
			}
			return x509.ParsePKCS1PrivateKey(b)
		}
		return x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	case pkcs8PrivateKeyPEMBlockType:
		priv, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
		if err != nil {
			return nil, err
		}
		switch priv := priv.(type) {
		case *rsa.PrivateKey:
			return priv, nil
		// case *dsa.PublicKey:
		// case *ecdsa.PublicKey:
		// case ed25519.PublicKey:
		default:
			panic("unsupported type of public key. SCEP need RSA private key")
		}
	default:
		return nil, errors.New("unmatched type or headers")
	}
}

// load an encrypted private key from disk
func loadCert(data []byte) (*x509.Certificate, error) {
	pemBlock, _ := pem.Decode(data)
	if pemBlock == nil {
		return nil, errors.New("PEM decode failed")
	}
	if pemBlock.Type != certificatePEMBlockType {
		return nil, errors.New("unmatched type or headers")
	}

	return x509.ParseCertificate(pemBlock.Bytes)
}

func pemCert(derBytes []byte) []byte {
	pemBlock := &pem.Block{
		Type:    certificatePEMBlockType,
		Headers: nil,
		Bytes:   derBytes,
	}
	out := pem.EncodeToMemory(pemBlock)
	return out
}
