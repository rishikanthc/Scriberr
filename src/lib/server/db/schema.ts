import { pgTable, index, jsonb, boolean, serial, text, integer, timestamp } from 'drizzle-orm/pg-core';
import { createId } from '@paralleldrive/cuid2';

export const user = pgTable('user', {
  id: text('id')
      .notNull()
      .primaryKey()
      .$defaultFn(() => createId()),
  username: text('username').notNull().unique(),
  passwordHash: text('password_hash').notNull(),
  isAdmin: boolean('is_admin').default(false).notNull(),
  createdAt: timestamp('created_at').defaultNow().notNull()
});

export const session = pgTable('session', {
  id: text('id').primaryKey(),
  userId: text('user_id')
    .notNull()
    .references(() => user.id),
  expiresAt: timestamp('expires_at', { withTimezone: true, mode: 'date' }).notNull()
});


export const systemSettings = pgTable('system_settings', {
  // Using serial instead of integer for auto-incrementing primary key
  id: serial('id').primaryKey(),
  isInitialized: boolean('is_initialized').notNull().default(false),
  firstStartupDate: timestamp('first_startup_date'),
  lastStartupDate: timestamp('last_startup_date'),
  whisperModelSizes: text('whisper_model_sizes').array(),
  whisperQuantization: text('whisper_quantization').notNull().default('none')
});

export const audioFiles = pgTable('audio_files', {
  id: serial('id').primaryKey(),
  
  // File information
  fileName: text('file_name').notNull(), // WAV file for transcription
  originalFileName: text('original_file_name'), // Original uploaded file name
  originalFileType: text('original_file_type'), // Original file format (mp3, etc.)
  duration: integer('duration'), // in seconds
  
  // Transcription
  transcript: jsonb('transcript'),
  transcriptionStatus: text('transcription_status', { 
    enum: ['pending', 'processing', 'completed', 'failed'] 
  }).default('pending').notNull(),
  
  // Summary
  summary: text('summary'),
  summaryPrompt: text('summary_prompt'),
  summaryStatus: text('summary_status', {
    enum: ['pending', 'processing', 'completed', 'failed']
  }),
  
  // Metadata & progress
  language: text('language').default('en'),
  lastError: text('last_error'),
  peaks: jsonb('peaks'),
  modelSize: text('model_size').notNull().default('base'),
  threads: integer('threads').notNull().default(4),
  title: text('title'),
  processors: integer('processors').notNull().default(1),
  diarization: boolean('diarization').default(false),
  transcriptionProgress: integer('transcription_progress').default(0),
  
  // Timestamps
  uploadedAt: timestamp('uploaded_at').defaultNow().notNull(),
  transcribedAt: timestamp('transcribed_at'),
  summarizedAt: timestamp('summarized_at'),
  updatedAt: timestamp('updated_at'),
}, (table) => {
  return {
    statusIdx: index('audio_files_status_idx').on(table.transcriptionStatus),
    uploadedAtIdx: index('audio_files_uploaded_at_idx').on(table.uploadedAt),
    summaryStatusIdx: index('audio_files_summary_status_idx').on(table.summaryStatus)
  };
});

export const speakerLabelsTable = pgTable('speaker_labels', {
  fileId: integer('file_id').primaryKey().references(() => audioFiles.id),
  labels: jsonb('labels').notNull(),
  createdAt: timestamp('created_at').defaultNow(),
  updatedAt: timestamp('updated_at').defaultNow()
});

export type TranscriptSegment = {
  start: number;
  end: number;
  text: string;
  speaker?: string;
};

export const summarizationTemplates = pgTable('summarization_templates', {
  id: text('id')
    .notNull()
    .primaryKey()
    .$defaultFn(() => createId()),
  title: text('title').notNull(),
  prompt: text('prompt').notNull(),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at').defaultNow()
}, (table) => {
  return {
    titleIdx: index('summarization_templates_title_idx').on(table.title)
  };
});

export type SummarizationTemplate = typeof summarizationTemplates.$inferSelect;

export type Session = typeof session.$inferSelect;

export type User = typeof user.$inferSelect;
