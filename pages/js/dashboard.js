const API_URL = "/system/utilization"
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
        const response = await fetch(API_URL)

        if (!response.ok) {
            throw new Error(`${response.status}: ${response.statusText}`)
        }

        const data = await response.json()

        document.getElementById("error").textContent = ""

        //
        // CPU
        //

        setText("cpu-mhz", `${data.cpu.mhz.toFixed(0)} MHz`)
        setText("cpu-overall", `${data.cpu.overallPercent.toFixed(1)}%`)
        setText("cpu-threads", `${data.cpu.threads.length}`)

        setBar("cpu-bar", data.cpu.overallPercent)

        //
        // Memory
        //

        setText("memory-omr", formatBytes(data.memory.inUseOmr))
        setText("memory-available", formatBytes(data.memory.totalAvailable))

        const memoryUsed =
            data.memory.inUseOmr +
            data.memory.inUseOther

        setText("memory-used", formatBytes(memoryUsed))
        setBar(
            "other-memory-bar",
            data.memory.inUseOther
                / (data.memory.totalAvailable+data.memory.inUseOther)
                * 100,
        )
        setBar(
            "omr-memory-bar",
            data.memory.inUseOmr
                / (data.memory.totalAvailable+data.memory.inUseOther)
                * 100,
        )

        //
        // Disk
        //

        setText("disk-used", formatBytes(data.disk.used))

        let omrUsage = 
            data.disk.database + 
            data.disk.matrices +
            data.disk.pictures
        setBar(
            "omr-disk-bar",
            (omrUsage / data.disk.total) * 100
        )
        setBar(
            "other-disk-bar",
            ((data.disk.used - omrUsage) / data.disk.total) * 100
        )

        setText("disk-available", formatBytes(data.disk.total - data.disk.used))
        setText("disk-omr-total", formatBytes(omrUsage))

        setText(
            "updated",
            `Updated ${new Date().toLocaleTimeString()}`
        )

        //
        // Update Peaks
        //

        peakOmrMemory = Math.max(peakOmrMemory, data.memory.inUseOmr)
        setText("omr-memory-peak", formatBytes(peakOmrMemory))

    } catch (err) {
        document.getElementById("error").textContent =
            `Failed to load stats: ${err.message}`
    }
}

update()
setInterval(update, INTERVAL)