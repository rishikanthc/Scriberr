<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { Switch } from '$lib/components/ui/switch';
	import { Label } from '$lib/components/ui/label';
	import * as Select from '$lib/components/ui/select';

	export let transcriptionOptions: {
		modelSize: 'tiny' | 'base' | 'small' | 'medium' | 'large' | 'large-v2' | 'large-v3';
		language: string;
		threads: number;
		processors: number;
		diarization: boolean;
	};

	const WHISPER_LANGUAGES = {
		auto: 'Auto Detect',
		af: 'Afrikaans',
		sq: 'Albanian',
		am: 'Amharic',
		ar: 'Arabic',
		hy: 'Armenian',
		as: 'Assamese',
		az: 'Azerbaijani',
		be: 'Belarusian',
		bn: 'Bengali',
		bs: 'Bosnian',
		bg: 'Bulgarian',
		ca: 'Catalan',
		zh: 'Chinese',
		hr: 'Croatian',
		cs: 'Czech',
		da: 'Danish',
		nl: 'Dutch',
		en: 'English',
		et: 'Estonian',
		fil: 'Filipino',
		fi: 'Finnish',
		fr: 'French',
		gl: 'Galician',
		ka: 'Georgian',
		de: 'German',
		el: 'Greek',
		gu: 'Gujarati',
		he: 'Hebrew',
		hi: 'Hindi',
		hu: 'Hungarian',
		is: 'Icelandic',
		id: 'Indonesian',
		ga: 'Irish',
		it: 'Italian',
		ja: 'Japanese',
		jw: 'Javanese',
		kn: 'Kannada',
		kk: 'Kazakh',
		km: 'Khmer',
		ko: 'Korean',
		lo: 'Lao',
		lv: 'Latvian',
		ln: 'Lingala',
		lt: 'Lithuanian',
		lb: 'Luxembourgish',
		mk: 'Macedonian',
		mg: 'Malagasy',
		ms: 'Malay',
		ml: 'Malayalam',
		mt: 'Maltese',
		mi: 'Maori',
		mr: 'Marathi',
		mn: 'Mongolian',
		my: 'Myanmar',
		ne: 'Nepali',
		no: 'Norwegian',
		or: 'Odia',
		ps: 'Pashto',
		fa: 'Persian',
		pl: 'Polish',
		pt: 'Portuguese',
		pa: 'Punjabi',
		ro: 'Romanian',
		ru: 'Russian',
		sa: 'Sanskrit',
		sr: 'Serbian',
		sd: 'Sindhi',
		si: 'Sinhala',
		sk: 'Slovak',
		sl: 'Slovenian',
		sn: 'Shona',
		so: 'Somali',
		es: 'Spanish',
		su: 'Sundanese',
		sw: 'Swahili',
		sv: 'Swedish',
		tl: 'Tagalog',
		tg: 'Tajik',
		ta: 'Tamil',
		tt: 'Tatar',
		te: 'Telugu',
		th: 'Thai',
		tk: 'Turkmen',
		tr: 'Turkish',
		ug: 'Uyghur',
		uk: 'Ukrainian',
		ur: 'Urdu',
		uz: 'Uzbek',
		vi: 'Vietnamese',
		cy: 'Welsh',
		yi: 'Yiddish',
		yo: 'Yoruba'
	};
</script>

<div class="mx-auto mt-6 w-[300px] space-y-4">
	<!-- Model Size -->
	<div class="space-y-2">
		<Label class="text-sm text-gray-100"
			>{$_('settings_panel.model_size.label')}
			<Select.Root bind:value={transcriptionOptions.modelSize} type="single">
				<Select.Trigger
					class="border border-neutral-500/40 bg-neutral-900/40 shadow-lg backdrop-blur-md"
					>{$_(`settings_panel.model_size.${transcriptionOptions.modelSize}`)}</Select.Trigger
				>
				<Select.Content>
					<Select.Item value="tiny">{$_('settings_panel.model_size.tiny')}</Select.Item>
					<Select.Item value="base">{$_('settings_panel.model_size.base')}</Select.Item>
					<Select.Item value="small">{$_('settings_panel.model_size.small')}</Select.Item>
					<Select.Item value="medium">{$_('settings_panel.model_size.medium')}</Select.Item>
					<Select.Item value="large">{$_('settings_panel.model_size.large')}</Select.Item>
					<Select.Item value="large-v2">{$_('settings_panel.model_size.large_v2')}</Select.Item>
					<Select.Item value="large-v3">{$_('settings_panel.model_size.large_v3')}</Select.Item>
				</Select.Content>
			</Select.Root>
		</Label>
	</div>

	<!-- Language -->
	<div class="space-y-2">
		<Label class="text-sm text-gray-100"
			>{$_('settings_panel.language.label')}
			<Select.Root bind:value={transcriptionOptions.language} type="single">
				<Select.Trigger
					class="border border-neutral-500/40 bg-neutral-900/40 shadow-lg backdrop-blur-md"
				>
					{transcriptionOptions.language === 'auto' ? $_('settings_panel.language.auto_detect') : WHISPER_LANGUAGES[transcriptionOptions.language]}
				</Select.Trigger>
				<Select.Content>
					{#each Object.entries(WHISPER_LANGUAGES) as [code, name]}
						<Select.Item value={code}>{code === 'auto' ? $_('settings_panel.language.auto_detect') : name}</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</Label>
	</div>

	<!-- CPU Threads -->
	<div class="space-y-2">
		<Label class="text-sm text-gray-100"
			>{$_('settings_panel.cpu_threads.label')}
			<Select.Root bind:value={transcriptionOptions.threads} type="single">
				<Select.Trigger
					class="border border-neutral-500/40 bg-neutral-900/40 shadow-lg backdrop-blur-md"
					>{transcriptionOptions.threads} {$_('settings_panel.cpu_threads.suffix')}</Select.Trigger
				>
				<Select.Content>
					{#each [1, 2, 4, 6, 8] as n}
						<Select.Item value={n}>{n} {$_('settings_panel.cpu_threads.suffix')}</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</Label>
	</div>

	<!-- CPU Processors -->
	<div class="space-y-2">
		<Label class="text-sm text-gray-100"
			>{$_('settings_panel.cpu_processors.label')}
			<Select.Root bind:value={transcriptionOptions.processors} type="single">
				<Select.Trigger
					class="border border-neutral-500/40 bg-neutral-900/40 shadow-lg backdrop-blur-md"
					>{transcriptionOptions.processors} {$_('settings_panel.cpu_processors.suffix')}</Select.Trigger
				>
				<Select.Content>
					{#each [1, 2, 3, 4] as n}
						<Select.Item value={n}>{n} {$_('settings_panel.cpu_processors.suffix')}</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</Label>
	</div>

	<!-- Diarization -->
	<div class="space-y-2">
		<Label class="text-sm font-bold text-gray-50">{$_('settings_panel.speaker_detection.label')}</Label>
		<div class="flex items-center space-x-2">
			<Switch
				id="diarization"
				class="data-[state=unchecked] data-[state=checked]:bg-blue-600 data-[state=unchecked]:bg-black"
				checked={transcriptionOptions.diarization}
				onCheckedChange={(checked) => (transcriptionOptions.diarization = checked)}
			/>
			<Label for="diarization" class="text-sm text-gray-100">{$_('settings_panel.speaker_detection.enable')}</Label>
		</div>
		{#if transcriptionOptions.diarization}
			<div class="mt-2 rounded-md bg-amber-800/20 p-2 text-xs text-amber-400">
				<p>
					<strong>{$_('settings_panel.speaker_detection.note_title')}</strong>
					{$_('settings_panel.speaker_detection.note_body')}
				</p>
			</div>
		{/if}
	</div>
</div>