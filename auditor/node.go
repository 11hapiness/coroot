package auditor

import (
	"fmt"

	"github.com/coroot/coroot/model"
	"github.com/coroot/coroot/timeseries"
	"github.com/coroot/coroot/utils"
	"github.com/dustin/go-humanize"
)

func AuditNode(w *model.World, node *model.Node) *model.AuditReport {
	report := model.NewAuditReport(nil, w.Ctx, nil, model.AuditReportNode, true)

	if !node.IsAgentInstalled() {
		return report
	}

	report.Status = model.OK

	report.AddWidget(&model.Widget{GroupHeader: "CPU", Width: "100%"})
	cpuByModeChart(
		report.GetOrCreateChart("CPU usage, %", model.NewDocLink("inspections", "cpu", "node-cpu-usage")),
		node.CpuUsageByMode,
	)

	ncs := getNodeConsumers(node)
	report.GetOrCreateChart("CPU consumers, cores", model.NewDocLink("inspections", "cpu", "cpu-consumers")).
		Stacked().
		Sorted().
		SetThreshold("total", node.CpuCapacity).
		AddMany(ncs.cpu, 5, timeseries.Max)

	used := timeseries.Sub(
		node.MemoryTotalBytes,
		timeseries.Sum(node.MemoryCachedBytes, node.MemoryFreeBytes),
	)
	report.AddWidget(&model.Widget{GroupHeader: "Memory", Width: "100%"})

	report.
		GetOrCreateChart("Memory usage, bytes", nil).
		Stacked().
		Sorted().
		AddSeries("free", node.MemoryFreeBytes, "light-blue").
		AddSeries("cache", node.MemoryCachedBytes, "amber").
		AddSeries("used", used, "red")

	report.GetOrCreateChart("Memory consumers, bytes", model.NewDocLink("inspections", "memory", "memory-consumers")).
		Stacked().
		Sorted().
		SetThreshold("total", node.MemoryTotalBytes).
		AddMany(ncs.memory, 5, timeseries.Max)

	report.AddWidget(&model.Widget{GroupHeader: "Network", Width: "100%"})

	for _, i := range node.NetInterfaces {
		report.
			GetOrCreateChartInGroup("Network bandwidth <selector>, bits/second", i.Name, nil).
			AddSeries("in", i.RxBytes.Map(func(t timeseries.Time, v float32) float32 { return v * 8 }), "green").
			AddSeries("out", i.TxBytes.Map(func(t timeseries.Time, v float32) float32 { return v * 8 }), "blue")
	}

	type vInfo struct {
		MountPoints   *utils.StringSet
		PVCs          *utils.StringSet
		CapacityBytes *timeseries.TimeSeries
		UsedBytes     *timeseries.TimeSeries
		Instances     *utils.StringSet
	}

	volumes := map[string]*vInfo{}
	for _, i := range node.Instances {
		for _, v := range i.Volumes {
			dev := v.Device.Value()
			if dev == "" {
				continue
			}
			vol := volumes[dev]
			if vol == nil {
				vol = &vInfo{
					MountPoints:   utils.NewStringSet(),
					CapacityBytes: v.CapacityBytes,
					UsedBytes:     v.UsedBytes,
					Instances:     utils.NewStringSet(),
					PVCs:          utils.NewStringSet(),
				}
				volumes[dev] = vol
			}
			vol.MountPoints.Add(v.MountPoint)
			vol.Instances.Add(i.Name)
			vol.PVCs.Add(v.Name.Value())
		}
	}
	if len(volumes) > 0 {
		report.AddWidget(&model.Widget{GroupHeader: "Disks", Width: "100%"})
	}
	disks := report.GetOrCreateTable("Device", "Mount points", "Used by", "Latency", "I/O Load", "Space")
	ioLatencyChart := report.GetOrCreateChartGroup("Average I/O latency <selector>, seconds", nil)
	ioLoadChart := report.GetOrCreateChartGroup("I/O load (total latency) <selector>, seconds/second", nil)
	iopsChart := report.GetOrCreateChartGroup("IOPS <selector>", nil)
	bandwidthChart := report.GetOrCreateChartGroup("Bandwidth <selector>, bytes/second", nil)
	spaceChart := report.GetOrCreateChartGroup("Disk space <selector>, bytes", nil)
	for device, d := range node.Disks {
		vol := volumes[device]
		if vol == nil {
			continue
		}

		ioLoad := timeseries.NewAggregate(timeseries.NanSum).Add(d.ReadTime, d.WriteTime).Get()
		ioLoadCell := model.NewTableCell()
		if v := ioLoad.Get().Last(); !timeseries.IsNaN(v) {
			ioLoadCell.SetValue(utils.FormatFloat(v))
		}
		space := model.NewTableCell()
		capacity := vol.CapacityBytes.Last()
		usage := vol.UsedBytes.Last()
		if usage > 0 && capacity > 0 {
			percentage := usage / capacity * 100
			space.SetValue(fmt.Sprintf(
				"%.0f%% (%s / %s)",
				percentage,
				humanize.Bytes(uint64(usage)),
				humanize.Bytes(uint64(capacity))),
			)
		}
		disks.AddRow(
			model.NewTableCell("/dev/"+device),
			model.NewTableCell(vol.MountPoints.Items()...),
			model.NewTableCell(vol.Instances.Items()...),
			model.NewTableCell().SetUnit("ms").SetValue(utils.FormatFloat(d.Await.Last()*1000)),
			ioLoadCell,
			space,
		)
		ioLatencyChart.GetOrCreateChart("overview").Feature().AddSeries(device, d.Await)
		ioLatencyChart.
			GetOrCreateChart(device).
			AddSeries("read", timeseries.Div(d.ReadTime, d.ReadOps), "blue").
			AddSeries("write", timeseries.Div(d.WriteTime, d.WriteOps), "amber")

		ioLoadChart.GetOrCreateChart("overview").Feature().AddSeries(device, ioLoad)
		ioLoadChart.GetOrCreateChart(device).
			Stacked().
			AddSeries("read", d.ReadTime, "blue").
			AddSeries("write", d.WriteTime, "amber")

		iopsChart.GetOrCreateChart("overview").Feature().
			AddSeries(device, timeseries.NewAggregate(timeseries.NanSum).Add(d.ReadOps, d.WriteOps))
		iopsChart.GetOrCreateChart(device).Stacked().Sorted().
			AddSeries("read", d.ReadOps, "blue").
			AddSeries("write", d.WriteOps, "amber")

		bandwidthChart.GetOrCreateChart(device).Stacked().Sorted().
			AddSeries("read", d.ReadBytes, "blue").
			AddSeries("written", d.WrittenBytes, "amber")

		spaceChart.GetOrCreateChart(device).Stacked().
			AddSeries("used", vol.UsedBytes).
			SetThreshold("total", vol.CapacityBytes)
	}

	if len(node.GPUs) > 0 {
		report.AddWidget(&model.Widget{GroupHeader: "GPUs", Width: "100%"})
	}

	for _, gpu := range node.GPUs {
		gpus := report.GetOrCreateTable("GPU UUID", "Name", "vRAM")
		mem := model.NewTableCell()
		if last := gpu.TotalMemory.Last(); last > 0 {
			mem.SetValue(humanize.Bytes(uint64(last)))
		}
		gpus.AddRow(model.NewTableCell(gpu.UUID), model.NewTableCell(gpu.Name.Value()), mem)
		report.
			GetOrCreateChartGroup("GPU utilization <selector>, %", nil).
			GetOrCreateChart("average").
			AddSeries(gpu.UUID, gpu.UsageAverage).Feature()
		report.
			GetOrCreateChartGroup("GPU utilization <selector>, %", nil).
			GetOrCreateChart("peak").
			AddSeries(gpu.UUID, gpu.UsagePeak)
		report.
			GetOrCreateChartGroup("GPU Memory utilization <selector>, %", nil).
			GetOrCreateChart("average").
			AddSeries(gpu.UUID, gpu.MemoryUsageAverage).Feature()
		report.
			GetOrCreateChartGroup("GPU Memory utilization <selector>, %", nil).
			GetOrCreateChart("peak").
			AddSeries(gpu.UUID, gpu.MemoryUsagePeak).Feature()
		coreChart := report.
			GetOrCreateChartGroup("GPU consumers <selector>, %", nil).
			GetOrCreateChart(gpu.UUID).Stacked()
		memChart := report.
			GetOrCreateChartGroup("GPU memory consumers <selector>, %", nil).
			GetOrCreateChart(gpu.UUID).Stacked()
		for _, ci := range gpu.Instances {
			if u := ci.GPUUsage[gpu.UUID]; u != nil {
				coreChart.AddSeries(ci.Name, u.UsageAverage)
				memChart.AddSeries(ci.Name, u.MemoryUsageAverage)
			}
		}
		report.
			GetOrCreateChart("GPU temperature, ℃", nil).
			AddSeries(gpu.UUID, gpu.Temperature)
		report.
			GetOrCreateChart("GPU power, watts", nil).
			AddSeries(gpu.UUID, gpu.PowerWatts)

	}
	return report
}
