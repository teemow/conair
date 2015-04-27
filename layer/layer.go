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
	Hash       string
	Verb       string
	Payload    string
	ParentId   string
	ParentPath string
	Path       string
	Exists     bool
	fs         *btrfs.Driver
}

func Create(fs *btrfs.Driver, verb, payload, parentPath string) (*layer, error) {
	l := &layer{
		Verb:       verb,
		Payload:    payload,
		ParentPath: parentPath,
		Exists:     false,
		fs:         fs,
	}
	var err error
	l.ParentId, err = l.getParentId()
	if err != nil {
		return nil, err
	}

	l.Hash, err = l.createHash()
	if err != nil {
		return nil, err
	}

	l.Path = fmt.Sprintf("conair/layers/%s", l.Hash)
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
	if strings.Index(l.ParentPath, "machines/") == 0 {
		parentId, err = l.fs.GetSubvolumeParentUuid(l.ParentPath)
	} else {
		parentId, err = l.fs.GetSubvolumeUuid(l.ParentPath)
	}
	if err != nil {
		return "", err
	}

	return parentId, nil
}

func (l *layer) createHash() (string, error) {
	h := sha1.New()

	io.WriteString(h, l.ParentId)
	io.WriteString(h, l.Verb)
	io.WriteString(h, l.Payload)

	if l.Verb == "ADD" {
		p := strings.Split(l.Payload, " ")
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
	if l.fs.Exists(l.Path) {
		if l.Verb == "RUN_NOCACHE" {
			if err := l.Remove(); err != nil {
				return fmt.Errorf("Couldn't remove existing layer. %v", err)
			}
		} else {
			l.Exists = true
			return nil
		}
	}

	if err := l.fs.Snapshot(l.ParentPath, l.Path, false); err != nil {
		return fmt.Errorf("Couldn't create filesystem for layer. %v", err)
	}
	return nil
}

func (l *layer) Remove() error {
	if err := l.fs.Remove(l.Path); err != nil {
		return fmt.Errorf("Couldn't remove layer. %v", err)
	}
	return nil
}
