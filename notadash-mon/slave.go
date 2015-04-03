package main

import (
    "os"
    "fmt"
    "github.com/codegangsta/cli"
    "github.com/ryanuber/columnize"
    lib "../lib"
)

var csRequired = []string{
    "marathon-host",
    "mesos-host",
}

func checkSlave(ctx *cli.Context) {
    if missing, err := validateContext(ctx, csRequired); err != nil {
        fmt.Println(err)
        fmt.Printf("The following parameters must be defined: %s\n", missing)
        os.Exit(2)
    }

    fmt.Println("Discoving running applications and associated tasks...")

    marathon := &lib.Marathon{
        Host: ctx.GlobalString("marathon-host"),
    }
    marathon.LoadApps()

    mesos := &lib.Mesos{
        Host: ctx.GlobalString("mesos-host"),
    }

    var host string
    if ip, err := lib.GetExternalIP(); err != nil {
        fmt.Println(err)
        fmt.Println("Unable to determin local host's IP, can't to proceed.")
        os.Exit(1)
    } else {
        host = fmt.Sprintf("%s", ip)
    }

    slave := mesos.LoadSlave(host)
    slaveFrameworks := slave.Framework("marathon")
    marathonApps := &lib.MarathonApps{}

    if len(slaveFrameworks) > 0 {
        for _, a := range marathon.Apps {
            if tasks, err := marathon.Client().Tasks(a.ID); err != nil {
                fmt.Println(err)
                os.Exit(1)
            } else {
                for _, t := range tasks.Tasks {
                    if slave.Slave.HostName == t.Host {
                        marathonApps.AddTask(t.ID, t.AppID, slave.Slave.Id, slave.Slave.HostName, false, true)
                    }
                }
            }
            for _, f := range slaveFrameworks {
                for _, e := range f.Executors {
                    for _, t := range e.Tasks {
                        mTask := marathonApps.AddTask(t.Id, t.AppId(), slave.Slave.Id, slave.Slave.HostName, true, false)
                        mTask.Container = e.RegisteredContainerName()
                    }
                }
            }
        }
    }

    output := make([]string, 1)
    output[0] = "Application | Task ID | Slave Host | Mesos/Marathon/Docker"
    discrepancy := false


    for _, a := range marathonApps.Apps {
        app_discrepancy := false
        app_output := make([]string, 1)
        app_output[0] = fmt.Sprintf("%s| | | ", a.Id)
        for _, t := range a.Tasks {
            var containerRunning = lib.ContainerRunning(t.Container)
            if !(t.Mesos && t.Marathon) || !containerRunning {
                app_discrepancy = true
                ln := fmt.Sprintf(
                    " | %s | %s | %s/%s/%s",
                    t.Id,
                    t.SlaveHost,
                    lib.PrintBool(t.Mesos),
                    lib.PrintBool(t.Marathon),
                    lib.PrintBool(containerRunning),
                )
                app_output = append(app_output, ln)
            }
        }
        if discrepancy = app_discrepancy; discrepancy {
            output = append(output, app_output...)
        }
    }

    if discrepancy {
        fmt.Println(lib.PrintYellow("Discrepency in task state found!"))
        result := columnize.SimpleFormat(output)
        fmt.Println(result)
        os.Exit(2)
    } else {
        fmt.Println(lib.PrintGreen("Mesos and Marathon agree about running tasks!"))
        os.Exit(0)
    }
}