package pids

import (
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
)

// FindPIDs goes over all running processes and finds the ones which cmdline matches the regex provided
func FindPIDs(r *regexp.Regexp) ([]int, error) {
	d, err := os.Open("/proc")
	if err != nil {
		return nil, err
	}
	defer d.Close()

	pids := []int{}

	fis, err := d.Readdir(-1)
	if err != nil {
		return nil, err
	}

	for _, fi := range fis {
		// We only care about directories, since all pids are dirs
		if !fi.IsDir() {
			continue
		}

		// We only care if the name starts with a numeric
		name := fi.Name()
		if name[0] < '0' || name[0] > '9' {
			continue
		}

		// From this point forward, any errors we just ignore, because
		// it might simply be that the process doesn't exist anymore.
		pid, err := strconv.Atoi(name)
		if err != nil {
			continue
		}

		f, err := ioutil.ReadFile("/proc/" + name + "/cmdline")
		if err != nil {
			continue
		}

		if r.Match(f) {
			pids = append(pids, pid)
		}
	}

	return pids, nil
}
