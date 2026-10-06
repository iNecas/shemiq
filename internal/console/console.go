package console

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var ErrCancelled = errors.New("selection cancelled")

// Console retains its buffered input across consecutive selections.
type Console struct {
	input  *bufio.Reader
	output io.Writer
}

func New(input io.Reader, output io.Writer) *Console {
	return &Console{input: bufio.NewReader(input), output: output}
}

// Pick prints a numbered menu and repeats the prompt until a valid choice or
// EOF (Ctrl-D). Blank and invalid lines are not cancellations.
func (c *Console) Pick(heading string, labels []string) (int, error) {
	if len(labels) == 0 {
		return 0, fmt.Errorf("no choices for %s", heading)
	}
	if _, err := fmt.Fprintln(c.output, heading); err != nil {
		return 0, err
	}
	for i, label := range labels {
		if _, err := fmt.Fprintf(c.output, "%d. %s\n", i+1, label); err != nil {
			return 0, err
		}
	}
	for {
		if _, err := fmt.Fprint(c.output, "Selection: "); err != nil {
			return 0, err
		}
		line, err := c.input.ReadString('\n')
		if err == io.EOF {
			return 0, ErrCancelled
		}
		if err != nil {
			return 0, fmt.Errorf("read selection: %w", err)
		}
		answer := strings.TrimSpace(line)
		n, parseErr := strconv.Atoi(answer)
		if answer != "" && parseErr == nil && n >= 1 && n <= len(labels) {
			return n - 1, nil
		}
		if _, err := fmt.Fprintf(c.output, "Please enter a number from 1 to %d.\n", len(labels)); err != nil {
			return 0, err
		}
	}
}
