package common

import (
	"fmt"
	"os"
	"runtime/pprof"
	"time"

	"github.com/shirou/gopsutil/cpu"
)

// Monitor 定时监控cpu使用率，超过阈值输出pprof文件
func Monitor() {
	for {
		percent, err := cpu.Percent(time.Second, false)
		if err != nil {
			SysLog("pprof monitor cpu.Percent failed: " + err.Error())
			time.Sleep(30 * time.Second)
			continue
		}
		if percent[0] > 80 {
			SysLog(fmt.Sprintf("cpu usage too high: %.2f%%", percent[0]))
			if _, err := os.Stat("./pprof"); os.IsNotExist(err) {
				err := os.Mkdir("./pprof", os.ModePerm)
				if err != nil {
					SysLog("创建pprof文件夹失败 " + err.Error())
					continue
				}
			}
			// cap pprof files to avoid filling disk (max 10 files)
			if entries, err := os.ReadDir("./pprof"); err == nil && len(entries) > 10 {
				SysLog(fmt.Sprintf("pprof directory has %d files, skip new profile", len(entries)))
				time.Sleep(30 * time.Second)
				continue
			}
			f, err := os.Create("./pprof/" + fmt.Sprintf("cpu-%s.pprof", time.Now().Format("20060102150405")))
			if err != nil {
				SysLog("创建pprof文件失败 " + err.Error())
				continue
			}
			err = pprof.StartCPUProfile(f)
			if err != nil {
				SysLog("启动pprof失败 " + err.Error())
				_ = f.Close()
				continue
			}
			time.Sleep(10 * time.Second)
			pprof.StopCPUProfile()
			_ = f.Close()
		}
		time.Sleep(30 * time.Second)
	}
}
