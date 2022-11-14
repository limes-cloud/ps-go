package main

import (
	"fmt"
	"log"
	"ps-go/engine"
	"ps-go/rooter"
	"ps-go/tools/hash"
	"ps-go/tools/pool"
	"runtime"
)

var beginMem runtime.MemStats

func main() {
	// 协程池初始化
	pool.Init()
	// 调度引擎初始化
	engine.Init()
	// 初始化hash一致性算法
	hash.Init()
	// api 初始化
	rg := rooter.Init()

	// 启动并监听端口
	if err := rg.Run(":8080"); err != nil {
		log.Fatalln(err)
	}
}

func PrintMemInfo(beginMem runtime.MemStats) {
	var endMem runtime.MemStats
	runtime.ReadMemStats(&endMem)

	fmt.Printf("\n已申请且仍在使用的字节数 Alloc = %v MiB", (endMem.Alloc-beginMem.Alloc)/1024/1024)
	fmt.Printf("\n已申请的总字节数 TotalAlloc = %v MiB", (endMem.TotalAlloc-beginMem.TotalAlloc)/1024/1024)
	fmt.Printf("\n从系统中获取的字节数 Sys = %v MiB", (endMem.Sys-beginMem.Sys)/1024/1024)
	fmt.Printf("\n指针查找的次数Lookups = %v", endMem.Lookups-beginMem.Lookups)
	fmt.Printf("\n申请内存的次数Mallocs = %v", endMem.Mallocs-beginMem.Mallocs)
	fmt.Printf("\n释放内存的次数Frees = %v", endMem.Frees-beginMem.Frees)
	fmt.Printf("\n当前协程数量 = %v", runtime.NumGoroutine())
}
