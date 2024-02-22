package bolt

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"math/big"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"

	"github.com/boltdb/bolt"
)

// Depot implements a SCEP certificate store using boltdb.
// https://github.com/boltdb/bolt
type Depot struct {
	*bolt.DB
}

const (
	certBucket = "scep_certificates"
)

// NewBoltDepot creates a depot.Depot backed by BoltDB.
func NewBoltDepot(db *bolt.DB) (*Depot, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(certBucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &Depot{db}, nil
}

// For some read operations Bolt returns a direct memory reference to
// the underlying mmap. This means that persistent references to these
// memory locations are volatile. Make sure to copy data for places we
// know references to this memeory will be kept.
func bucketGetCopy(b *bolt.Bucket, key []byte) (out []byte) {
	in := b.Get(key)
	if in == nil {
		return
	}
	out = make([]byte, len(in))
	copy(out, in)
	return
}

func (db *Depot) CA(pass []byte) ([]*x509.Certificate, *rsa.PrivateKey, error) {
	chain := []*x509.Certificate{}
	var key *rsa.PrivateKey
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(certBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found!", certBucket)
		}
		// get ca_certificate
		caCert := bucketGetCopy(bucket, []byte("ca_certificate"))
		if caCert == nil {
			return errors.New("no ca_certificate in bucket")
		}
		cert, err := x509.ParseCertificate(caCert)
		if err != nil {
			return err
		}
		chain = append(chain, cert)

		// get ca_key
		caKey := bucket.Get([]byte("ca_key"))
		if caKey == nil {
			return errors.New("no ca_key in bucket")
		}
		key, err = x509.ParsePKCS1PrivateKey(caKey)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return chain, key, nil
}

func (db *Depot) Put(cn string, crt *x509.Certificate) error {
	if crt == nil || crt.Raw == nil {
		return fmt.Errorf("%q does not specify a valid certificate for storage", cn)
	}
	serial, err := db.Serial()
	if err != nil {
		return err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(certBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found!", certBucket)
		}
		name := cn + "." + serial.String()
		return bucket.Put([]byte(name), crt.Raw)
	})
	if err != nil {
		return err
	}
	return db.incrementSerial(serial)
}

func (db *Depot) Serial() (*big.Int, error) {
	s := big.NewInt(2)
	if !db.hasKey([]byte("serial")) {
		if err := db.writeSerial(s); err != nil {
			return nil, err
		}
		return s, nil
	}
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(certBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found!", certBucket)
		}
		k := bucket.Get([]byte("serial"))
		if k == nil {
			return fmt.Errorf("key %q not found", "serial")
		}
		s = s.SetBytes(k)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (db *Depot) writeSerial(s *big.Int) error {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(certBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found!", certBucket)
		}
		return bucket.Put([]byte("serial"), s.Bytes())
	})
	return err
}

func (db *Depot) hasKey(name []byte) bool {
	var present bool
	_ = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(certBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found!", certBucket)
		}
		k := bucket.Get([]byte("serial"))
		if k != nil {
			present = true
		}
		return nil
	})
	return present
}

func (db *Depot) incrementSerial(s *big.Int) error {
	serial := s.Add(s, big.NewInt(1))
	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(certBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found!", certBucket)
		}
		return bucket.Put([]byte("serial"), serial.Bytes())
	})
	return err
}

func (db *Depot) HasCN(cn string, allowTime int, cert *x509.Certificate, revokeOldCertificate bool) (bool, error) {
	// TODO: implement allowTime
	// TODO: implement revocation
	if cert == nil {
		return false, errors.New("nil certificate provided")
	}
	var hasCN bool
	err := db.View(func(tx *bolt.Tx) error {
		// TODO: "scep_certificates" is internal const in micromdm/scep
		curs := tx.Bucket([]byte("scep_certificates")).Cursor()
		prefix := []byte(cert.Subject.CommonName)
		for k, v := curs.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = curs.Next() {
			if bytes.Compare(v, cert.Raw) == 0 {
				hasCN = true
				return nil
			}
		}

		return nil
	})
	return hasCN, err
}

func (db *Depot) CreateOrLoadKey(bits int) (*rsa.PrivateKey, error) {
	var (
		key *rsa.PrivateKey
		err error
	)
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(certBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found!", certBucket)
		}
		priv := bucket.Get([]byte("ca_key"))
		if priv == nil {
			return nil
		}
		key, err = x509.ParsePKCS1PrivateKey(priv)
		return err
	})
	if err != nil {
		return nil, err
	}
	if key != nil {
		return key, nil
	}
	key, err = rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(certBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found!", certBucket)
		}
		return bucket.Put([]byte("ca_key"), x509.MarshalPKCS1PrivateKey(key))
	})
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (db *Depot) CreateOrLoadCA(key *rsa.PrivateKey, years int, org, country string) (*x509.Certificate, error) {
	var (
		cert *x509.Certificate
		err  error
	)
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(certBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found!", certBucket)
		}
		caCert := bucketGetCopy(bucket, []byte("ca_certificate"))
		if caCert == nil {
			return nil
		}
		cert, err = x509.ParseCertificate(caCert)
		return err
	})
	if err != nil {
		return nil, err
	}
	if cert != nil {
		return cert, nil
	}

	newCert := depot.NewCACert(
		depot.WithYears(years),
		depot.WithOrganization(org),
		depot.WithOrganizationalUnit("MICROMDM SCEP CA"),
		depot.WithCountry(country),
	)
	crtBytes, err := newCert.SelfSign(rand.Reader, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(certBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found!", certBucket)
		}
		return bucket.Put([]byte("ca_certificate"), crtBytes)
	})
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(crtBytes)
}
