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

function formatTime(seconds) {
    let minutes = Math.floor(seconds / 60)
    seconds %= 60
    let hours = Math.floor(minutes / 60)
    minutes %= 60
    let days = Math.floor(hours / 60)
    hours %= 60

    let secStr = (seconds < 10 ? "0" : "") + seconds.toString()
    let minStr = (minutes < 10 ? "0" : "") + minutes.toString()
    let hourStr = (hours < 10 ? "0" : "") + hours.toString()
    let dayStr = (days < 10 ? "0" : "") + days.toString()

    return dayStr + ":" + hourStr + ":" + minStr + ":" + secStr
}

function formatLogs(logs) {
    return logs
        .split("\n")
        .slice(-11, -1)
        .map(s => {
            s = s.replaceAll(" ", "&nbsp;")
            if (s.includes("[LOG]")) {
                return `<span class="misc">${s}</span>`
            } else if (s.includes("[INFO]")) {
                return `<span class="info">${s}</span>`
            } else if (s.includes("[WARN]")) {
                return `<span class="warn">${s}</span>`
            } else if (s.includes("[DEBUG]")) {
                return `<span class="debug">${s}</span>`
            } else if (s.includes("[ERROR]")) {
                return `<span class="error">${s}</span>`
            } else {
                return `<span class="misc">${s}</span>`
            }
        })
        .join("<br />") + "<br />"
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
        setText("memory-free", formatBytes(memory.free))

        const memoryUsed =
            memory.inUseOmr +
            memory.inUseOther

        const totalMemory =
            memory.free +
            memory.inUseOther +
            memory.inUseOmr

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
        // Misc
        //

        setText(
            "omr-memory-peak",
            formatBytes(data.memoryPeak)
        )
        setText(
            "uptime",
            formatTime(data.uptime)
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
        let logsOutput = document.getElementById("logs-output")
        logsOutput.innerHTML = formatLogs(data)
        logsOutput.scrollTo(0, logsOutput.scrollHeight)

    } catch (err) {
        document.getElementById("error").textContent =
            `Failed to load stats: ${err.message}`
    }
}

update()
setInterval(update, INTERVAL)
setInterval(updateLogs, INTERVAL)