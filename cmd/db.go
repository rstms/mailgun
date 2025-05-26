package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type DB struct {
	path    string
	verbose bool
	mutex   sync.Mutex
}

func NewDB(dir, name string) *DB {

	if strings.HasPrefix(dir, "~") {
		_, dir, _ = strings.Cut(dir, "~")
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("NewDB: %v", err)
		}
		dir = filepath.Join(home, dir)
	}
	if !IsDir(dir) {
		err := os.Mkdir(dir, 0700)
		if err != nil {
			log.Fatalf("NewDB: %v", err)
		}
	}
	path := filepath.Join(dir, name)
	if !IsDir(path) {
		err := os.Mkdir(path, 0700)
		if err != nil {
			log.Fatalf("NewDB: %v", err)
		}
	}
	db := DB{path: path, verbose: viper.GetBool("verbose")}
	if db.verbose {
		log.Printf("DB: dir=%s\n", db.path)
	}
	return &db
}

func (d *DB) Reset() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	err := os.RemoveAll(d.path)
	if err != nil {
		return err
	}
	err = os.Mkdir(d.path, 0700)
	if err != nil {
		return err
	}
	if viper.GetBool("verbose") {
		log.Printf("DB: reset %s\n", d.path)
	}
	return nil
}

func (d *DB) pathname(key string) string {
	filename := base64.RawURLEncoding.EncodeToString([]byte(key))
	return filepath.Join(d.path, filename)
}

func (d *DB) key(pathname string) (string, error) {
	_, filename := filepath.Split(pathname)
	decoded, err := base64.RawURLEncoding.DecodeString(filename)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func (d *DB) Has(key string) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	pathname := d.pathname(key)
	ret := IsFile(pathname)
	if d.verbose {
		log.Printf("DB.Has(%s) returning %v\n", key, ret)
	}
	return ret
}

func (d *DB) GetObject(key string, object any) (bool, error) {
	data, err := d.Get(key)
	if err != nil {
		return false, err
	}
	if data == nil {
		return false, fmt.Errorf("key not found: %s", key)
	}
	err = json.Unmarshal(*data, object)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (d *DB) Get(key string) (*[]byte, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	pathname := d.pathname(key)
	if !IsFile(pathname) {
		return nil, nil
	}
	data, err := os.ReadFile(pathname)
	if err != nil {
		return nil, err
	}
	if d.verbose {
		log.Printf("DB.Get(%s) read %d bytes from %s\n", key, len(data), pathname)
	}
	return &data, nil
}

func (d *DB) SetObject(key string, object interface{}) error {
	data, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		return err
	}
	return d.Set(key, &data)
}

func (d *DB) Set(key string, data *[]byte) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	pathname := d.pathname(key)
	err := os.WriteFile(pathname, *data, 0600)
	if err != nil {
		return err
	}
	if d.verbose {
		log.Printf("DB.Set(%s) wrote %d bytes to %s\n", key, len(*data), pathname)
	}
	return nil
}

func (d *DB) Clear(key string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	pathname := d.pathname(key)
	err := os.Remove(pathname)
	if err != nil {
		return err
	}
	if d.verbose {
		log.Printf("DB.Clear(%s) deleted %s\n", key, pathname)
	}
	return nil
}

func (d *DB) Keys() ([]string, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	keys := []string{}
	err := filepath.WalkDir(d.path, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == d.path {
			return nil
		}
		if entry.IsDir() {
			return fs.SkipDir
		}
		key, err := d.key(path)
		if err != nil {
			return err
		}
		keys = append(keys, key)
		return nil
	})
	if err != nil {
		return []string{}, err
	}
	if d.verbose {
		log.Printf("DB.Keys() returning %d keys\n", len(keys))
	}
	return keys, nil
}
