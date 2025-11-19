import Window from "./Window";

type Item = {
	title: string;
	desc: string;
	img: string;
	bullets?: string[];
};

const items: Item[] = [
	{
		title: "Transcript view",
		desc: "Minimal reading experience with timestamps with playback follow along that highlights currently playing word.",
		img: "scriberr-transcript page.png",
		bullets: [
			"Jump from audio timestamp to corresponding word",
			"Jump from text to corresponding audio segment",
			"View transcript as paragraph or as timestamped/speaker segments",
		],
	},
	{
		title: "Record right in Scriberr",
		desc: "Capture audio directly in-app and transcribe",
		img: "scriberr-inbuilt audio recorder for directly recording and transcribing audio within the app.png",
		bullets: [],
	},
	{
		title: "Summaries at a glance",
		desc: "Turn long recordings into brief, actionable summaries you can scan in seconds.",
		img: "scriberr-summarize transcripts.png",
		bullets: [
			"Write your own custom prompts for summarization",
			"Supports both Ollama/OpenAI (needs API Key) LLM providers",
			"Save multiple summarization presets to reuse quickly",
		],
	},
	{
		title: "Annotate transcripts",
		desc: "Highlight important moments, jot down concise notes, and keep insights attached to the exact timestamp.",
		img: "scriberr-annotate transcript and take notes.png",
		bullets: [
			"Highlight text to add a note",
			"Timestamped notes allow jumping to exact segment",
		],
	},
	{
		title: "Advanced controls",
		desc: "Fine-tune model settings, language, and diarization for optimal results",
		img: "scriberr-fine tune advanced transcription parameters as you see fit to improve transcription quality.png",
		bullets: [
			"Language hints and temperature",
			"Diarization and VAD options",
			"Profiles for repeatable setups",
		],
	},
	{
		title: "Bring your own providers",
		desc: "Use OpenAI or local models via Ollama for summaries and chat — your keys, your choice.",
		img: "scriberr-Ollama:openAI llm providers for chat and summarization.png",
		bullets: ["Works with OpenAI or Ollama"],
	},
	{
		title: "Export transcripts",
		desc: "Download your transcripts in multiple formats",
		img: "scriberr-download-transcript-in-different-formats.png",
		bullets: [
			"Export to TXT, Markdown, JSON",
			"Keep timestamps and speaker info",
		],
	},
	{
		title: "Chat with your transcript",
		desc: "Ask questions about your recording, extract insights, and clarify details without scrubbing through audio.",
		img: "scriberr-chat-with-your-recording-transcript.png",
		bullets: ["Works with OpenAI or Ollama"],
	},
	{
		title: "API keys and REST API",
		desc: "Manage API keys and use the full REST API to build automations or integrate Scriberr into your own applications.",
		img: "scriberr-api-key-management.png",
		bullets: [
			"Secure API key management",
			"Endpoints for transcription, chat, notes and more",
		],
	},
	{
		title: "Transcribe YouTube videos",
		desc: "Paste a YouTube link to transcribe the audio directly — no downloads required.",
		img: "scriberr-youtube-video.png",
		bullets: [
			"Grab insights from talks, podcasts and lectures",
			"Works with summarization and notes",
		],
	},
];

export default function Alternating() {
	return (
		<section id="details" className="container-narrow section">
			<div className="text-center mb-12">
				<span className="eyebrow">Details</span>
				<h2 className="text-2xl md:text-3xl font-semibold mt-2">
					A closer look
				</h2>
			</div>

			<div className="space-y-16 md:space-y-24">
				{items.map((item, i) => (
					<div
						key={item.title}
						className="grid md:grid-cols-2 gap-8 md:gap-10 items-center"
					>
						{/* On md+, alternate: even -> text left, image right; odd -> image left, text right */}
						<div
							className={
								i % 2 === 0 ? "order-1 md:order-1" : "order-2 md:order-2"
							}
						>
							<div>
								<h3 className="text-xl md:text-2xl font-semibold text-gray-900">
									{item.title}
								</h3>
								<p className="subcopy mt-2">{item.desc}</p>
								{item.bullets && (
									<ul className="mt-4 space-y-2 text-sm text-gray-600 list-disc list-inside">
										{item.bullets.map((b) => (
											<li key={b}>{b}</li>
										))}
									</ul>
								)}
							</div>
						</div>
						<div
							className={
								i % 2 === 0 ? "order-2 md:order-2" : "order-1 md:order-1"
							}
						>
							<Window src={`/screenshots/${item.img}`} alt={item.title} />
						</div>
					</div>
				))}
			</div>
		</section>
	);
}
