import { ScriberrLogo } from "./ScriberrLogo";

export default function Hero() {
	return (
		<section className="relative">
			<div className="container-narrow section text-center">
				<span className="eyebrow mb-3 inline-block">
					Self-hosted offline audio transcription
				</span>
				<h1 className="headline">
					<ScriberrLogo className="text-5xl sm:text-6xl md:text-7xl font-normal" />
				</h1>
				<p className="subcopy mt-3 mx-auto max-w-2xl">
					Transcribe audio locally into text - Summarize and Chat with your
					audio. No GPU Required.
				</p>
				<div className="mt-8 flex items-center justify-center gap-3">
					<a href="/docs/installation.html" className="button-primary">
						Get Started
					</a>
					<a href="#features" className="button-ghost">
						Learn more
					</a>
				</div>
				<div className="mt-6 flex justify-center">
					<a 
						href='https://ko-fi.com/H2H41KQZA3' 
						target='_blank' 
						rel="noopener noreferrer"
						onClick={(e) => {
							e.preventDefault();
							window.open('https://ko-fi.com/H2H41KQZA3', '_blank', 'noopener,noreferrer');
						}}
					>
						<img height='36' style={{border: '0px', height: '36px'}} src='https://storage.ko-fi.com/cdn/kofi6.png?v=6' alt='Buy Me a Coffee at ko-fi.com' />
					</a>
				</div>
				<div className="mt-12 max-w-5xl mx-auto">
					<div className="rounded-3xl shadow-soft overflow-hidden bg-white hover-lift">
						<div className="flex items-center gap-2 px-3 py-2 bg-gray-100">
							<span className="size-3 rounded-full bg-red-400/80" />
							<span className="size-3 rounded-full bg-yellow-400/80" />
							<span className="size-3 rounded-full bg-green-400/80" />
						</div>
						<img
							src="/screenshots/scriberr-homepage.png"
							alt="Scriberr homepage"
							className="w-full object-cover"
						/>
					</div>
				</div>
				<div className="mt-6 flex items-center justify-center gap-4 text-xs text-gray-500">
					<span>Privacy preserving</span>
					<span>Mobile ready</span>
					<span>Fast and responsive</span>
				</div>
			</div>
		</section>
	);
}
