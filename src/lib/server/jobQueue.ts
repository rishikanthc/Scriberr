import type { TranscribeStream } from './transcribeStream';

interface Job {
    id: number;
    streams: Set<TranscribeStream>;
    isRunning: boolean;
}

class JobQueue {
    private jobs: Map<number, Job> = new Map();

    addJob(id: number) {
        if (!this.jobs.has(id)) {
            this.jobs.set(id, {
                id,
                streams: new Set(),
                isRunning: false
            });
        }
        return this.jobs.get(id)!;
    }

    addStream(id: number, stream: TranscribeStream) {
        const job = this.jobs.get(id);
        if (job) {
            job.streams.add(stream);
            return true;
        }
        return false;
    }

    removeStream(id: number, stream: TranscribeStream) {
        const job = this.jobs.get(id);
        if (job) {
            job.streams.delete(stream);
            // Clean up job if no streams are listening
            if (job.streams.size === 0 && !job.isRunning) {
                this.jobs.delete(id);
            }
        }
    }

    setJobRunning(id: number, running: boolean) {
        const job = this.jobs.get(id);
        if (job) {
            job.isRunning = running;
            // Clean up completed jobs with no listeners
            if (!running && job.streams.size === 0) {
                this.jobs.delete(id);
            }
        }
    }

    getJob(id: number) {
        return this.jobs.get(id);
    }

    broadcastToJob(id: number, data: any) {
        const job = this.jobs.get(id);
        if (job) {
            job.streams.forEach(stream => {
                stream.sendProgress(data).catch(console.error);
            });
        }
    }
}

// Create a singleton instance
export const jobQueue = new JobQueue();
