package output

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"

	"github.com/Ensono/taskctl/pkg/task"
)

var frame = 100 * time.Millisecond
var base *baseCockpit

type baseCockpit struct {
	w       io.Writer
	tasks   []*task.Task
	mu      sync.Mutex
	spinner *spinner.Spinner
	charSet int
	closeCh chan bool
}

type cockpitOutputDecorator struct {
	b *baseCockpit
	t *task.Task
}

func (b *baseCockpit) start() *spinner.Spinner {
	if b.spinner != nil {
		return b.spinner
	}

	s := spinner.New(spinner.CharSets[b.charSet], frame, spinner.WithColor("yellow"))
	s.Writer = b.w
	s.PreUpdate = func(s *spinner.Spinner) {
		tasks := make([]string, 0)
		b.mu.Lock()
		defer b.mu.Unlock()
		for _, v := range b.tasks {
			tasks = append(tasks, v.Name)
		}
		sort.Strings(tasks)
		s.Suffix = " Running: " + strings.Join(tasks, ", ")
	}
	s.Start()

	return s
}

func (b *baseCockpit) add(t *task.Task) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.tasks = append(b.tasks, t)

	if b.spinner == nil {
		b.spinner = b.start()
		go func() {
			<-b.closeCh
			b.spinner.Stop()
		}()
	}
}

func (b *baseCockpit) remove(t *task.Task) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for k, v := range b.tasks {
		if v == t {
			b.tasks = append(b.tasks[:k], b.tasks[k+1:]...)
		}
	}

	var mark = fmt.Sprintf("\x1b[32m%s\x1b[0m", "✔")
	if t.Errored {
		mark = fmt.Sprintf("\x1b[31m%s\x1b[0m", "✗")
	}
	b.spinner.FinalMSG = fmt.Sprintf("%s Finished %s in %s\r\n", mark, fmt.Sprintf("\x1b[1m%s", t.Name), t.Duration())
	b.spinner.Restart()
	b.spinner.FinalMSG = ""
}

func NewCockpitOutputWriter(t *task.Task, w io.Writer, close chan bool) *cockpitOutputDecorator {
	return &cockpitOutputDecorator{
		t: t,
		b: &baseCockpit{
			charSet: 14,
			w:       NewSafeWriter(w),
			tasks:   make([]*task.Task, 0),
			closeCh: close,
		},
	}
}

func (d *cockpitOutputDecorator) Write(p []byte) (int, error) {
	return len(p), nil
}

func (d *cockpitOutputDecorator) WriteHeader() error {
	d.b.add(d.t)
	return nil
}

func (d *cockpitOutputDecorator) WriteFooter() error {
	d.b.remove(d.t)
	return nil
}
