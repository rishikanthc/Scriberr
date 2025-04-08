//lib/server/transcribeStream.ts
export class TranscribeStream {
  private writer: WritableStreamDefaultWriter | null = null;
  private closed = false;
  private bufferedMessages: any[] = [];

  constructor(writer?: WritableStreamDefaultWriter) {
    if (writer) {
      this.writer = writer;
      // If there are any buffered messages, send them now
      if (this.bufferedMessages.length > 0) {
        this.flushBufferedMessages();
      }
    }
  }

  setWriter(writer: WritableStreamDefaultWriter) {
    this.writer = writer;
    // If there are any buffered messages, send them now
    if (this.bufferedMessages.length > 0) {
      this.flushBufferedMessages();
    }
  }

  private async flushBufferedMessages() {
    if (this.writer && this.bufferedMessages.length > 0) {
      for (const data of this.bufferedMessages) {
        await this.writer.write(`data: ${JSON.stringify(data)}\n\n`);
      }
      this.bufferedMessages = [];
    }
  }

  async sendProgress(data: any) {
    if (this.closed) return;
    
    if (this.writer) {
      await this.writer.write(`data: ${JSON.stringify(data)}\n\n`);
    } else {
      // Buffer the message for when a writer is set
      this.bufferedMessages.push(data);
    }
  }

  async close() {
    if (!this.closed) {
      this.closed = true;
      if (this.writer) {
        await this.writer.close();
      }
    }
  }
}