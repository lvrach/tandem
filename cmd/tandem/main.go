package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	b64 "encoding/base64"

	"github.com/spaolacci/murmur3"
	"gopkg.in/yaml.v2"
)

type Lock struct {
	Files map[string]File
}

type File struct {
	Hash string
	Tags []string
}

func computeFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	buf := make([]byte, 1000)
	hasher := murmur3.New128()

	for {
		n, err := f.Read(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}
		_, err = hasher.Write(buf[:n])
		if err != nil {
			return "", err
		}
	}

	return "m3_128:" + b64.StdEncoding.EncodeToString(hasher.Sum([]byte{})), nil
}

func loadLock() (*Lock, error) {
	l := &Lock{
		Files: make(map[string]File),
	}

	d, err := ioutil.ReadFile(".tandem-lock.yaml")
	if err != os.ErrNotExist {
		return l, nil
	} else if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(d, l)
	if err != nil {
		return nil, err
	}

	return l, nil
}

func saveLock(l *Lock) error {
	d, err := yaml.Marshal(l)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(".tandem-lock.yaml", d, 0644)
}

func lock(paths []string, tags []string) {
	l, err := loadLock()
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range paths {
		h, err := computeFileHash(p)
		if err != nil {
			log.Fatal(err)
		}

		f := l.Files[p]
		if f.Hash == h {
			// tags := f.Tags
			// tags = append(tags, tags...)
			// f.Tags = tags
			continue
		}
		l.Files[p] = File{
			Hash: h,
			Tags: tags,
		}
	}

	err = saveLock(l)
	if err != nil {
		log.Fatal(err)
	}
}

func check(paths []string, tags []string) {
	l, err := loadLock()
	if err != nil {
		log.Fatal(err)
	}

	modifiedFiles := []string{}

	for _, p := range paths {
		h, err := computeFileHash(p)
		if err != nil {
			log.Fatal(err)
		}

		f := l.Files[p]
		if f.Hash != h {
			modifiedFiles = append(modifiedFiles, p)
		}
	}

	if len(modifiedFiles) > 0 {
		fmt.Printf("changed with out tandem lock:\n %s\n", strings.Join(modifiedFiles, "/n"))
		os.Exit(1)
	}
}

func main() {

	if len(os.Args) < 2 {
		panic("missing params")
	}
	switch os.Args[1] {
	case "lock":
		lock(os.Args[2:], []string{"default"})
	case "check":
		check(os.Args[2:], []string{"default"})
	}
}
