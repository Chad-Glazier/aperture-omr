const INTERVAL = 1000

let peakOmrMemory = 0

function formatBytes(bytes) {
    const units = ["B", "KB", "MB", "GB", "TB"]

    let value = bytes
    let unit = 0

    while (value >= 1024 && unit < units.length - 1) {
        value /= 1024
        unit++
    }

    return `${value.toFixed(2)} ${units[unit]}`
}

function setText(id, value) {
    document.getElementById(id).textContent = value
}

function setBar(id, percent) {
    document.getElementById(id).style.width =
        `${Math.min(100, Math.max(0, percent))}%`
}

async function update() {
    try {
        const response = await fetch("/system/utilization")

        if (!response.ok) {
            throw new Error(`${response.status}: ${response.statusText}`)
        }

        const data = await response.json()

        const cpu = data.cpuHistory[data.cpuHistory.length - 1]
        const memory = data.memoryHistory[data.memoryHistory.length - 1]
        const disk = data.disk

        //
        // CPU
        //

        setText("cpu-mhz", `${cpu.mhz.toFixed(0)} MHz`)
        setText("cpu-overall", `${cpu.overallPercent.toFixed(1)}%`)
        setText("cpu-threads", `${cpu.threads.length}`)

        setBar("cpu-bar", cpu.overallPercent)

        //
        // Memory
        //

        setText("memory-omr", formatBytes(memory.inUseOmr))
        setText("memory-available", formatBytes(memory.totalAvailable))

        const memoryUsed =
            memory.inUseOmr +
            memory.inUseOther

        const totalMemory =
            memory.totalAvailable +
            memory.inUseOther

        setText("memory-used", formatBytes(memoryUsed))

        setBar(
            "other-memory-bar",
            (memory.inUseOther / totalMemory) * 100,
        )

        setBar(
            "omr-memory-bar",
            (memory.inUseOmr / totalMemory) * 100,
        )

        //
        // Disk
        //

        setText("disk-used", formatBytes(disk.usage.used))

        const omrUsage = disk.omrUsage.total

        setBar(
            "omr-disk-bar",
            (omrUsage / disk.usage.total) * 100
        )

        setBar(
            "other-disk-bar",
            ((disk.usage.used - omrUsage) / disk.usage.total) * 100
        )

        setText(
            "disk-available",
            formatBytes(disk.usage.free)
        )

        setText(
            "disk-omr-total",
            formatBytes(omrUsage)
        )

        //
        // Update Peaks
        //

        peakOmrMemory = Math.max(
            peakOmrMemory,
            memory.inUseOmr,
        )

        setText(
            "omr-memory-peak",
            formatBytes(peakOmrMemory)
        )

    } catch (err) {
        alert(`Failed to load stats: ${err.message}`)
    }
}

async function updateLogs() {
    try {
        const response = await fetch("/system/logs")

        if (!response.ok) {
            throw new Error(`${response.status}: ${response.statusText}`)
        }

        let data = await response.text()
        data = data.split("\n").slice(-10).join("\n")

        setText("logs-output", data)

        let logsContainer = document.getElementById("logs-output")
        logsContainer.scrollTo(0, logsContainer.scrollHeight)

    } catch (err) {
        document.getElementById("error").textContent =
            `Failed to load stats: ${err.message}`
    }
}

update()
setInterval(update, INTERVAL)
setInterval(updateLogs, INTERVAL)