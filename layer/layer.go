package layer

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/giantswarm/conair/btrfs"
)

type layer struct {
	hash       string
	verb       string
	payload    string
	parentId   string
	parentPath string
	path       string
	fs         *btrfs.Driver
}

func Create(fs *btrfs.Driver, verb, payload, parentPath string) (*layer, error) {
	l := &layer{
		verb:       verb,
		payload:    payload,
		parentPath: parentPath,
		fs:         fs,
	}
	l.parentId, err = l.getParentId()
	if err != nil {
		return nil, err
	}

	l.hash, err = l.createHash()
	if err != nil {
		return nil, err
	}

	l.path = fmt.Sprintf("layers/%s", l.hash)
	if err = l.createLayer(); err != nil {
		return nil, err
	}

	return l, nil
}

func (l *layer) getParentId() (string, error) {
	var (
		parentId string
		err      error
	)
	if strings.Index(l.parentPath, "images/") == 0 {
		parentId, err = l.fs.GetSubvolumeParentUuid(l.parentPath)
	} else {
		parentId, err = l.fs.GetSubvolumeUuid(l.parentPath)
	}
	if err != nil {
		return "", err
	}

	return parentId, nil
}

func (l *layer) createHash() (string, error) {
	h := sha1.New()

	io.WriteString(h, l.parentId)
	io.WriteString(h, l.verb)
	io.WriteString(h, l.payload)

	if l.verb == "ADD" {
		p := strings.Split(l.payload, " ")
		sourceFile := p[0]

		f, err := os.Open(sourceFile)
		if err != nil {
			return "", err
		}
		defer f.Close()
		reader := bufio.NewReader(f)

		_, err = io.Copy(h, reader)
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (l *layer) createLayer() error {
	if l.fs.Exists(l.path) {
		if l.verb == "RUN_NOCACHE" {
			if err := l.Remove(); err != nil {
				return fmt.Errorf("Couldn't remove existing layer. %v", err)
			}
		} else {
			return nil
		}
	}

	if err := l.fs.Snapshot(l.parentPath, l.path, false); err != nil {
		return fmt.Errorf("Couldn't create filesystem for layer. %v", err)
	}
	return nil
}

func (l *layer) Remove() error {
	if err := l.fs.Remove(l.path); err != nil {
		return fmt.Errorf("Couldn't remove layer. %v", err)
	}
	return nil
}
