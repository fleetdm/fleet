package bolt

import (
	"io/ioutil"
	"math/big"
	"os"
	"reflect"
	"testing"

	bolt "go.etcd.io/bbolt"
)

// createDepot creates a Bolt database in a temporary location.
func createDB(mode os.FileMode, options *bolt.Options) *Depot {
	// Create temporary path.
	f, _ := ioutil.TempFile("", "bolt-")
	f.Close()
	os.Remove(f.Name())

	db, err := bolt.Open(f.Name(), mode, options)
	if err != nil {
		panic(err.Error())
	}
	d, err := NewBoltDepot(db)
	if err != nil {
		panic(err.Error())
	}
	return d
}

func TestDepot_Serial(t *testing.T) {
	db := createDB(0o666, nil)
	tests := []struct {
		name    string
		want    *big.Int
		wantErr bool
	}{
		{
			name: "two is the default value.",
			want: big.NewInt(2),
		},
	}
	for _, tt := range tests {
		got, err := db.Serial()
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. Depot.Serial() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. Depot.Serial() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestDepot_writeSerial(t *testing.T) {
	db := createDB(0o666, nil)

	tests := []struct {
		name    string
		args    *big.Int
		wantErr bool
	}{
		{
			args: big.NewInt(5),
		},
		{
			args: big.NewInt(3),
		},
	}
	for _, tt := range tests {
		if err := db.writeSerial(tt.args); (err != nil) != tt.wantErr {
			t.Errorf("%q. Depot.writeSerial() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestDepot_incrementSerial(t *testing.T) {
	db := createDB(0o666, nil)

	tests := []struct {
		name    string
		args    *big.Int
		want    *big.Int
		wantErr bool
	}{
		{
			args: big.NewInt(2),
			want: big.NewInt(3),
		},
		{
			args: big.NewInt(3),
			want: big.NewInt(4),
		},
	}
	for _, tt := range tests {
		if err := db.incrementSerial(tt.args); (err != nil) != tt.wantErr {
			t.Errorf("%q. Depot.incrementSerial() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
		got, _ := db.readSerial()
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. Depot.Serial() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestDepot_CreateOrLoadKey(t *testing.T) {
	db := createDB(0o666, nil)
	tests := []struct {
		bits    int
		wantErr bool
	}{
		{
			bits: 1024,
		},
		{
			bits: 2048,
		},
	}
	for i, tt := range tests {
		if _, err := db.CreateOrLoadKey(tt.bits); (err != nil) != tt.wantErr {
			t.Errorf("%d. Depot.CreateOrLoadKey() error = %v, wantErr %v", i, err, tt.wantErr)
		}
	}
}

func TestDepot_CreateOrLoadCA(t *testing.T) {
	db := createDB(0o666, nil)
	tests := []struct {
		wantErr bool
	}{
		{},
		{},
	}
	for i, tt := range tests {
		key, err := db.CreateOrLoadKey(1024)
		if err != nil {
			t.Fatalf("%d. Depot.CreateOrLoadKey() error = %v", i, err)
		}

		if _, err := db.CreateOrLoadCA(key, 10, "MicroMDM", "US"); (err != nil) != tt.wantErr {
			t.Errorf("%d. Depot.CreateOrLoadCA() error = %v, wantErr %v", i, err, tt.wantErr)
		}
	}
}
