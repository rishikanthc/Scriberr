const features = [
	{
		title: "Precise transcription",
		desc: "Tweak advanced transcription parameters to get the best quality output",
	},
	{
		title: "Built-in recorder",
		desc: "Capture audio directly in-app and transcribe instantly.",
	},
	{
		title: "Summarize & chat",
		desc: "Extract key points or chat over transcripts using LLMs.",
	},
	{
		title: "Lightweight notes",
		desc: "Highlight, annotate, and tag important moments as you listen/read.",
	},
	{
		title: "Speaker diarization",
		desc: "Identify and label distinct speakers in your audio.",
	},
	{
		title: "Profiles & presets",
		desc: "Save configurations for different audio scenarios.",
	},
];

function Icon({ name }: { name: string }) {
	const common = "size-4";
	switch (name) {
		case "Precise transcription":
			return (
				<svg
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					strokeWidth="1.8"
					className={common}
				>
					<path d="M12 3v10a3 3 0 1 1-6 0V8" />
					<path d="M19 10v3a7 7 0 0 1-14 0" />
				</svg>
			);
		case "Built-in recorder":
			return (
				<svg
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					strokeWidth="1.8"
					className={common}
				>
					<rect x="9" y="9" width="6" height="6" rx="3" />
					<circle cx="12" cy="12" r="9" />
				</svg>
			);
		case "Summarize & chat":
			return (
				<svg
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					strokeWidth="1.8"
					className={common}
				>
					<path d="M21 15a4 4 0 0 1-4 4H7l-4 3V7a4 4 0 0 1 4-4h10a4 4 0 0 1 4 4z" />
				</svg>
			);
		case "Lightweight notes":
			return (
				<svg
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					strokeWidth="1.8"
					className={common}
				>
					<path d="M9 7h6M9 12h6M9 17h6" />
					<rect x="5" y="3" width="14" height="18" rx="2" />
				</svg>
			);
		case "Speaker diarization":
			return (
				<svg
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					strokeWidth="1.8"
					className={common}
				>
					<circle cx="7" cy="8" r="3" />
					<path d="M2 19a5 5 0 0 1 10 0" />
					<circle cx="17" cy="10" r="2.5" />
					<path d="M13 19c.5-2.5 2.5-4 5-4" />
				</svg>
			);
		case "Profiles & presets":
			return (
				<svg
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					strokeWidth="1.8"
					className={common}
				>
					<path d="M6 3v12" />
					<path d="M12 3v18" />
					<path d="M18 3v8" />
				</svg>
			);
		default:
			return (
				<svg
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					strokeWidth="1.8"
					className={common}
				>
					<path d="M12 6v12m6-6H6" />
				</svg>
			);
	}
}

export default function Features() {
	return (
		<section id="features" className="container-narrow section">
			<div className="text-center mb-12">
				<span className="eyebrow">Capabilities</span>
				<h2 className="text-2xl md:text-3xl font-semibold mt-2">
					Key Features
				</h2>
				<p className="subcopy mt-2">
					Curated set of features to manage and work with transcripts.
				</p>
			</div>

			<div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-6">
				{features.map((f) => (
					<article
						key={f.title}
						className="rounded-2xl p-6 bg-gray-50 shadow-soft hover-lift"
					>
						<div className="mb-3 flex items-center gap-3">
							<span className="inline-flex items-center justify-center size-9 rounded-xl bg-gray-100 text-gray-600 shadow-subtle">
								<Icon name={f.title} />
							</span>
							<h3 className="font-medium text-base md:text-lg text-gray-900">
								{f.title}
							</h3>
						</div>
						<p className="subcopy">{f.desc}</p>
					</article>
				))}
			</div>
		</section>
	);
}
