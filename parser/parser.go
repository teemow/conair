package parser

import (
	"bufio"
	"os"
	"strings"
)

type Command struct {
	Verb    string
	Payload string
}

type Conairfile struct {
	From     string
	Commands []Command
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

func Parse(path string) (*Conairfile, error) {
	lines, err := readFile(path)
	if err != nil {
		return nil, err
	}

	d := &Conairfile{}
	for _, line := range lines {
		if !strings.HasPrefix(line, "#") {
			l := strings.SplitN(line, " ", 2)
			if len(l) > 1 {
				cmd := Command{
					Verb:    l[0],
					Payload: l[1],
				}

				switch cmd.Verb {
				case "FROM":
					d.From = cmd.Payload
				case "ADD":
					d.Commands = append(d.Commands, cmd)
				case "RUN":
					d.Commands = append(d.Commands, cmd)
				case "PKG":
					d.Commands = append(d.Commands, cmd)
				case "ENABLE":
					d.Commands = append(d.Commands, cmd)
				}
			}
		}
	}
	return d, nil
}
