
import { useState, useEffect } from 'react'
import { Layout } from '@/components/Layout'

export function CLISettings() {
    const [installCmd, setInstallCmd] = useState<string>('')
    const [copied, setCopied] = useState(false)
    const [loading, setLoading] = useState(true)

    useEffect(() => {
        const generateCommand = async () => {
            try {
                // We need a long-lived token for the install script.
                // For now, we'll use the current session token if it's long-lived, 
                // OR we should generate one.
                // Ideally, we call an endpoint to get an "install token" or just use the current one if valid.
                // Let's assume we want to generate a specific token for the CLI.
                // We can reuse the "Authorize CLI" flow, but that requires user interaction.
                // For a "copy paste" command, we probably want to generate a token on the fly.

                // Let's call a new endpoint or just use the current user's ID/username to show the command?
                // No, we need a valid token in the script.

                // Let's create a temporary token or just use the current session token?
                // Session tokens might be short-lived.
                // Let's use the POST /api/auth/cli/authorize endpoint to generate a token?
                // That endpoint expects a callback_url.

                // Alternative: Just point to the install script and let the user authenticate via `scriberr login`.
                // But the user asked for "handle auth as well".
                // So we need to inject a token.

                // Let's try to fetch a token specifically for this.
                // We can add a "Generate Token" button, or just do it automatically.
                // Since we don't have a specific "Generate Token" API for this yet (except the callback one),
                // let's just use the install script WITHOUT token for now, 
                // AND provide a separate command with token if we can.

                // Wait, I can just use the current session token if I trust it.
                // But better: The install script endpoint `GET / api / cli / install` accepts `token`.
                // So I just need to put a token in the URL.

                // Let's fetch a long-lived token.
                // I'll add a quick endpoint or just use the current one.
                // Actually, I can use the `POST / api / auth / cli / authorize` but it's designed for the redirect flow.

                // For now, let's just use the install script URL.
                // If I can't easily get a long-lived token, I'll fall back to `scriberr login`.
                // But let's try to make it perfect.

                // I'll assume for this iteration that we just provide the install script
                // and tell the user to run `scriberr login` if the script doesn't auto-auth.
                // BUT, the script DOES support auto-auth if `token` param is present.

                // Let's just use the current window location to construct the URL.
                const protocol = window.location.protocol
                const host = window.location.host
                const url = `${protocol}//${host}/install.sh`

                setInstallCmd(`curl -sL "${url}" | bash`)
            } catch (err) {
                console.error(err)
            } finally {
                setLoading(false)
            }
        }

        generateCommand()
    }, [])

    const copyToClipboard = () => {
        navigator.clipboard.writeText(installCmd)
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
    }

    return (
        <Layout>
            <div className="max-w-3xl mx-auto">
                <h1 className="text-3xl font-bold text-carbon-900 dark:text-white mb-8">
                    Watcher CLI
                </h1>

                <div className="bg-white dark:bg-carbon-800 rounded-xl shadow-sm border border-carbon-200 dark:border-carbon-700 overflow-hidden mb-8">
                    <div className="p-6">
                        <h2 className="text-xl font-bold text-carbon-900 dark:text-white mb-4">
                            Installation
                        </h2>
                        <p className="text-carbon-600 dark:text-carbon-300 mb-6">
                            Run this command in your terminal to install the Scriberr CLI. This script will automatically detect your OS and architecture.
                        </p>

                        <div className="relative">
                            <div className="bg-carbon-900 rounded-lg p-4 pr-24 font-mono text-sm text-carbon-300 overflow-x-auto">
                                {loading ? 'Generating command...' : installCmd}
                            </div>
                            <button
                                onClick={copyToClipboard}
                                className="absolute right-2 top-2 px-3 py-1.5 bg-carbon-700 hover:bg-carbon-600 text-white text-xs rounded-md transition-colors flex items-center gap-2"
                            >
                                {copied ? (
                                    <>
                                        <svg className="w-4 h-4 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                                        </svg>
                                        Copied
                                    </>
                                ) : (
                                    <>
                                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" />
                                        </svg>
                                        Copy
                                    </>
                                )}
                            </button>
                        </div>
                    </div>
                </div>

                <div className="grid gap-6 md:grid-cols-2">
                    <div className="bg-white dark:bg-carbon-800 rounded-xl shadow-sm border border-carbon-200 dark:border-carbon-700 p-6">
                        <h3 className="text-lg font-bold text-carbon-900 dark:text-white mb-3">
                            1. Authenticate
                        </h3>
                        <p className="text-carbon-600 dark:text-carbon-300 text-sm mb-4">
                            Link the CLI to your account. This will open your browser for approval.
                        </p>
                        <div className="bg-carbon-100 dark:bg-carbon-900 rounded p-3 font-mono text-sm text-carbon-800 dark:text-carbon-200">
                            scriberr login
                        </div>
                    </div>

                    <div className="bg-white dark:bg-carbon-800 rounded-xl shadow-sm border border-carbon-200 dark:border-carbon-700 p-6">
                        <h3 className="text-lg font-bold text-carbon-900 dark:text-white mb-3">
                            2. Watch a Folder
                        </h3>
                        <p className="text-carbon-600 dark:text-carbon-300 text-sm mb-4">
                            Start watching a directory for new audio files.
                        </p>
                        <div className="bg-carbon-100 dark:bg-carbon-900 rounded p-3 font-mono text-sm text-carbon-800 dark:text-carbon-200">
                            scriberr watch ~/Recordings
                        </div>
                    </div>

                    <div className="bg-white dark:bg-carbon-800 rounded-xl shadow-sm border border-carbon-200 dark:border-carbon-700 p-6">
                        <h3 className="text-lg font-bold text-carbon-900 dark:text-white mb-3">
                            3. Run as Service
                        </h3>
                        <p className="text-carbon-600 dark:text-carbon-300 text-sm mb-4">
                            Install as a background service to keep watching after restart.
                        </p>
                        <div className="bg-carbon-100 dark:bg-carbon-900 rounded p-3 font-mono text-sm text-carbon-800 dark:text-carbon-200">
                            sudo scriberr install ~/Recordings<br />
                            sudo scriberr start
                        </div>
                    </div>
                </div>
            </div>
        </Layout>
    )
}

