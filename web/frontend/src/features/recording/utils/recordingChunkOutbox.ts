export type StoredRecordingChunk = {
  recordingId: string;
  chunkIndex: number;
  blob: Blob;
  mimeType: string;
  durationMs?: number;
};

const databaseName = "scriberr-recording-outbox";
const databaseVersion = 1;
const storeName = "chunks";

export async function storeRecordingChunk(chunk: StoredRecordingChunk): Promise<void> {
  const db = await openOutbox();
  await requestToPromise(
    transactionStore(db, "readwrite").put({
      ...chunk,
      id: chunkKey(chunk.recordingId, chunk.chunkIndex),
      createdAt: Date.now(),
    })
  );
  db.close();
}

export async function deleteRecordingChunk(recordingId: string, chunkIndex: number): Promise<void> {
  const db = await openOutbox();
  await requestToPromise(transactionStore(db, "readwrite").delete(chunkKey(recordingId, chunkIndex)));
  db.close();
}

export async function listRecordingChunks(): Promise<StoredRecordingChunk[]> {
  const db = await openOutbox();
  const rows = await requestToPromise<Array<StoredRecordingChunk & { id: string; createdAt: number }>>(
    transactionStore(db, "readonly").getAll()
  );
  db.close();
  return rows
    .sort((a, b) => a.createdAt - b.createdAt || a.chunkIndex - b.chunkIndex)
    .map(({ recordingId, chunkIndex, blob, mimeType, durationMs }) => ({
      recordingId,
      chunkIndex,
      blob,
      mimeType,
      durationMs,
    }));
}

function openOutbox(): Promise<IDBDatabase> {
  if (!("indexedDB" in window)) return Promise.reject(new Error("IndexedDB is unavailable"));

  return new Promise((resolve, reject) => {
    const request = indexedDB.open(databaseName, databaseVersion);
    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(storeName)) {
        db.createObjectStore(storeName, { keyPath: "id" });
      }
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error || new Error("Could not open recording chunk outbox"));
  });
}

function transactionStore(db: IDBDatabase, mode: IDBTransactionMode) {
  return db.transaction(storeName, mode).objectStore(storeName);
}

function requestToPromise<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error || new Error("IndexedDB request failed"));
  });
}

function chunkKey(recordingId: string, chunkIndex: number) {
  return `${recordingId}:${chunkIndex}`;
}
