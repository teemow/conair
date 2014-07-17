package parser

import (
	"bufio"
	"os"
	"strings"
)

type command struct {
	Verb    string
	Payload string
}

type dockerfile struct {
	From     string
	Commands []command
}

func readFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func Dockerfile(path string) (*dockerfile, error) {
	lines, err := readFile(path)
	if err != nil {
		return nil, err
	}

	d := &dockerfile{}
	for _, line := range lines {
		if !strings.HasPrefix(line, "#") {
			l := strings.SplitN(line, " ", 2)
			if len(l) > 1 {
				cmd := command{
					Verb:    l[0],
					Payload: l[1],
				}

				switch cmd.Verb {
				case "FROM":
					d.From = cmd.Payload
				default:
					d.Commands = append(d.Commands, cmd)
				}
			}
		}
	}
	return d, nil
}
