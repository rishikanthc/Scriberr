//lib/server/transcribeStream.ts
export class TranscribeStream {
  private writer: WritableStreamDefaultWriter;
  private closed = false;

  constructor(writer: WritableStreamDefaultWriter) {
    this.writer = writer;
  }

  async sendProgress(data: any) {
    if (!this.closed) {
      await this.writer.write(`data: ${JSON.stringify(data)}\n\n`);
    }
  }

  async close() {
    if (!this.closed) {
      this.closed = true;
      await this.writer.close();
    }
  }
}
