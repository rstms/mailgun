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
)

type DB struct {
	path    string
	verbose bool
}

func isDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func isFile(pathname string) bool {
	_, err := os.Stat(pathname)
	return !os.IsNotExist(err)
}

func NewDB(name, dir string) (*DB, error) {
	var err error
	if dir == "" {
		dir, err = os.UserCacheDir()
		if err != nil {
			return nil, err
		}
	} else if strings.HasPrefix(dir, "~") {
		_, dir, _ = strings.Cut(dir, "~")
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(home, dir)
	}
	path := filepath.Join(dir, name)
	if !isDir(path) {
		err := os.Mkdir(path, 0700)
		if err != nil {
			return nil, err
		}
	}
	db := DB{path: path, verbose: viper.GetBool("verbose")}
	if db.verbose {
		log.Printf("NewDB: directory=%s\n", db.path)
	}
	return &db, nil
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
	pathname := d.pathname(key)
	ret := isFile(pathname)
	if d.verbose {
		log.Printf("DB.Has(%s) filename=%s returning %v\n", key, pathname, ret)
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
	pathname := d.pathname(key)
	if !isFile(pathname) {
		return nil, nil
	}
	data, err := os.ReadFile(pathname)
	if err != nil {
		return nil, err
	}
	if d.verbose {
		log.Printf("DB.Get(%s) filename=%s read %s\n", key, pathname, string(data))
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

	pathname := d.pathname(key)
	err := os.WriteFile(pathname, *data, 0600)
	if err != nil {
		return err
	}
	if d.verbose {
		log.Printf("DB.Set(%s) filename=%s wrote %s\n", key, pathname, string(*data))
	}
	return nil
}

func (d *DB) Keys() ([]string, error) {

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
		log.Printf("DB.Keys() returning %v\n", keys)
	}
	return keys, nil
}
