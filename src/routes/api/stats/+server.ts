import { json } from '@sveltejs/kit';
import os from 'os';

export function GET() {
	// Get system memory usage
	const totalMemory = os.totalmem();
	const freeMemory = os.freemem();
	const usedMemory = totalMemory - freeMemory;

	// Get CPU usage
	const cpus = os.cpus();
	const cpuLoad = cpus.map((cpu) => {
		const { user, nice, sys, idle, irq } = cpu.times;
		const total = user + nice + sys + idle + irq;
		return {
			model: cpu.model,
			speed: cpu.speed,
			usage: ((total - idle) / total) * 100
		};
	});

	return json({
		memory: {
			total: totalMemory,
			used: usedMemory,
			free: freeMemory,
			usedPercentage: (usedMemory / totalMemory) * 100
		},
		cpu: cpuLoad
	});
}
