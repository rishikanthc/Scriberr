
import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import {
    Zap, Shield, Mic, Cpu, Layers, MessageSquare,
    FileText, Video, Terminal, Globe, FolderOpen, Notebook
} from 'lucide-react';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Section } from '../components/ui/Section';
import { Heading, Paragraph, GradientText } from '../components/ui/Typography';

export function Home() {
    const container = {
        hidden: { opacity: 0 },
        show: {
            opacity: 1,
            transition: { staggerChildren: 0.1 }
        }
    };

    return (
        <div className="space-y-12">
            {/* Hero Section */}
            <Section className="!py-12 sm:py-10 min-h-[auto] md:min-h-[80vh] flex flex-col items-center justify-center text-center">
                <motion.div
                    initial={{ opacity: 0, y: 30 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.8, ease: "easeOut" }}
                    className="space-y-6 max-w-4xl mx-auto"
                >
                    <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-orange-50 border border-orange-100 text-orange-600 text-sm font-medium">
                        <span className="relative flex h-2 w-2">
                            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-orange-400 opacity-75"></span>
                            <span className="relative inline-flex rounded-full h-2 w-2 bg-orange-500"></span>
                        </span>
                        v1.2.0 Now Available
                    </div>

                    <Heading level={1}>
                        Transcribe Everything.<br />
                        <GradientText>Privately.</GradientText>
                    </Heading>

                    <Paragraph size="lg" className="max-w-2xl mx-auto">
                        A self-hostable offline audio transcription app.
                        State-of-the-art AI models, running entirely on your machine.
                    </Paragraph>

                    <div className="flex items-center justify-center pt-4">
                        <Link to="/docs/intro">
                            <Button size="lg" className="px-8">Get Started</Button>
                        </Link>
                    </div>
                </motion.div>

                {/* Hero Image Showcase */}
                <motion.div
                    initial={{ opacity: 0, y: 50 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 1, delay: 0.2 }}
                    className="relative w-full max-w-6xl mx-auto mt-8 md:mt-20 perspective-1000"
                >
                    {/* Gradient Glow */}
                    <div className="absolute inset-0 bg-[image:var(--image-brand-gradient)] opacity-20 rounded-[2rem] blur-3xl -z-10 transform scale-90"></div>

                    <div className="glass-panel rounded-2xl overflow-hidden p-2">
                        <img
                            src="/screenshots/transcript-light-2.png"
                            alt="Scriberr Interface"
                            className="w-full h-auto rounded-xl shadow-inner bg-white"
                        />
                    </div>

                    <motion.div
                        animate={{ y: [0, -10, 0] }}
                        transition={{ duration: 6, repeat: Infinity, ease: "easeInOut" }}
                        className="hidden md:block absolute -bottom-12 -right-12 w-[300px] rounded-[2.5rem] border-8 border-gray-900 overflow-hidden shadow-2xl bg-white"
                    >
                        <img
                            src="/screenshots/mobile-transcript-light.PNG"
                            alt="Mobile Interface"
                            className="w-full h-full object-cover"
                        />
                    </motion.div>
                </motion.div>

                {/* Sponsor Section */}
                <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    transition={{ delay: 1.2, duration: 1 }}
                    className="mt-20 flex flex-col items-center justify-center space-y-4"
                >
                    <span className="text-xs font-semibold text-gray-400 uppercase tracking-[0.2em]">Sponsors</span>
                    <a
                        href="https://www.recall.ai/?utm_source=github&utm_medium=sponsorship&utm_campaign=rishikanthc-scriberr"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="group transition-all duration-300 hover:-translate-y-0.5"
                    >
                        <img
                            src="https://cdn.prod.website-files.com/620d732b1f1f7b244ac89f0e/66b294e51ee15f18dd2b171e_recall-logo.svg"
                            alt="Recall.ai"
                            className="h-7 md:h-8 w-auto hover:opacity-80 transition-opacity duration-300"
                        />
                    </a>
                </motion.div>
            </Section>

            {/* Features Grid */}
            <Section id="features" className="bg-gray-50/50">
                <div className="text-center mb-16 space-y-4">
                    <Heading level={2}>Your audio, transcribed on your terms.</Heading>
                    <Paragraph className="max-w-2xl mx-auto">
                        Get accurate text, speaker labels, and AI summaries without ever sending your data to the cloud.
                    </Paragraph>
                </div>

                <motion.div
                    variants={container}
                    initial="hidden"
                    whileInView="show"
                    viewport={{ once: true, margin: "-100px" }}
                    className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8"
                >
                    {features.map((feature, index) => (
                        <Card key={index} className="h-full group">
                            <div className="w-12 h-12 rounded-xl bg-orange-50 text-orange-600 flex items-center justify-center mb-6 group-hover:scale-110 transition-transform duration-300">
                                {feature.icon}
                            </div>
                            <Heading level={4} className="mb-3">{feature.title}</Heading>
                            <Paragraph size="sm">
                                {feature.description}
                            </Paragraph>
                        </Card>
                    ))}
                </motion.div>
            </Section>
        </div>
    );
}

const features = [
    {
        icon: <Cpu className="w-6 h-6" />,
        title: "NVIDIA & Whisper Models",
        description: "Choose between state-of-the-art NVIDIA Parakeet/Canary models or OpenAI's Whisper. Optimized for accuracy and speed."
    },
    {
        icon: <Shield className="w-6 h-6" />,
        title: "100% Private & Local",
        description: "All processing happens securely on your device. Your audio data never leaves your machine."
    },
    {
        icon: <Zap className="w-6 h-6" />,
        title: "Hardware Acceleration",
        description: "Full support for NVIDIA GPUs with CUDA, plus optimized CPU inference for maximum performance."
    },
    {
        icon: <Layers className="w-6 h-6" />,
        title: "Speaker Diarization",
        description: "Automatically detect and label distinct speakers in your recordings for clear, structured transcripts."
    },
    {
        icon: <MessageSquare className="w-6 h-6" />,
        title: "AI Chat & Summary",
        description: "Chat with your transcripts and generate summaries using Ollama or OpenAI compatible providers."
    },
    {
        icon: <Mic className="w-6 h-6" />,
        title: "In-Built Recorder",
        description: "Record audio directly within the app with high-fidelity capture and instant transcription."
    },
    {
        icon: <Video className="w-6 h-6" />,
        title: "YouTube Transcription",
        description: "Paste a YouTube link to instantly download and transcribe video content."
    },
    {
        icon: <Globe className="w-6 h-6" />,
        title: "PWA Support",
        description: "Install as a native-feeling app on any device with Progressive Web App capabilities."
    },
    {
        icon: <FileText className="w-6 h-6" />,
        title: "Karaoke-style Playback",
        description: "Follow along with synchronized word-by-word highlighting during audio playback."
    },
    {
        icon: <FolderOpen className="w-6 h-6" />,
        title: "Directory Watcher",
        description: "Automatically upload and transcribe recordings as soon as they appear in a monitored folder."
    },
    {
        icon: <Notebook className="w-6 h-6" />,
        title: "Note Taking",
        description: "Jot down important points, timestamps, and ideas alongside your transcriptions."
    },
    {
        icon: <Terminal className="w-6 h-6" />,
        title: "Developer API",
        description: "Extensive API support and webhooks for building powerful automation workflows."
    }
];
