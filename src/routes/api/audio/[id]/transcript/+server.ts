import { error, json } from '@sveltejs/kit';
import { db } from '$lib/server/db';
import { eq } from 'drizzle-orm';
import { audioFiles, speakerLabelsTable } from '$lib/server/db/schema';
import { requireAuth } from '$lib/server/auth';

export async function PUT({ params, request, locals }) {
  await requireAuth(locals);

  try {
    const { id } = params;
    const fileId = parseInt(id);
    const { transcript, speakerLabels } = await request.json();
    
    const updatedTranscript = transcript.map(segment => ({
      ...segment,
      speaker: segment.speaker ? speakerLabels[segment.speaker] || segment.speaker : undefined
    }));
    
    await db.transaction(async (tx) => {
      await tx.delete(speakerLabelsTable)
        .where(eq(speakerLabelsTable.fileId, fileId));
      
      await tx.insert(speakerLabelsTable)
        .values({
          fileId,
          labels: speakerLabels,
          updatedAt: new Date()
        });

      await tx.update(audioFiles)
        .set({
          'transcript': JSON.stringify(updatedTranscript),
          'updated_at': new Date()
        })
        .where(eq(audioFiles.id, fileId));
    });

    return json({ success: true, transcript: updatedTranscript });
  } catch (err) {
    console.error('Update failed:', err);
    throw error(500, 'Failed to update transcript');
  }
}
