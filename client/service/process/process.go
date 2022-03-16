package process

import "github.com/shirou/gopsutil/v3/process"

type Process struct {
	Name string `json:"name"`
	Pid  int32  `json:"pid"`
}

func ListProcesses() ([]Process, error) {
	result := make([]Process, 0)
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(processes); i++ {
		name, err := processes[i].Name()
		if err != nil {
			name = `<Unknown>`
		}
		result = append(result, Process{Name: name, Pid: processes[i].Pid})
	}
	return result, nil
}

func KillProcess(pid int32) error {
	processes, err := process.Processes()
	if err != nil {
		return err
	}
	for i := 0; i < len(processes); i++ {
		if processes[i].Pid == pid {
			return processes[i].Kill()
		}
	}
	return nil
}
