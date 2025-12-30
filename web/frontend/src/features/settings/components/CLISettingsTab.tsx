import { useState, useEffect } from 'react'


export function CLISettingsTab() {
    const [installCmd, setInstallCmd] = useState<string>('')
    const [copied, setCopied] = useState(false)
    const [loading, setLoading] = useState(true)

    useEffect(() => {
        const generateCommand = async () => {
            try {
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
        <div className="space-y-6">
            <div className="bg-[var(--bg-main)]/50 rounded-[var(--radius-card)] shadow-sm border border-[var(--border-subtle)] overflow-hidden">
                <div className="p-6">
                    <h2 className="text-xl font-bold text-[var(--text-primary)] mb-4">
                        Installation
                    </h2>
                    <p className="text-[var(--text-secondary)] mb-6">
                        Run this command in your terminal to install the Scriberr CLI. This script will automatically detect your OS and architecture.
                    </p>

                    <div className="relative">
                        <div className="bg-[#0f172a] rounded-lg p-4 pr-24 font-mono text-sm text-gray-300 overflow-x-auto border border-[var(--border-subtle)]">
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
                <div className="bg-[var(--bg-main)]/50 rounded-[var(--radius-card)] shadow-sm border border-[var(--border-subtle)] p-6">
                    <h3 className="text-lg font-bold text-[var(--text-primary)] mb-3">
                        1. Authenticate
                    </h3>
                    <p className="text-[var(--text-secondary)] text-sm mb-4">
                        Link the CLI to your account. This will open your browser for approval.
                    </p>
                    <div className="bg-[var(--bg-card)] rounded p-3 font-mono text-sm text-[var(--text-primary)] border border-[var(--border-subtle)]">
                        scriberr login
                    </div>
                </div>

                <div className="bg-[var(--bg-main)]/50 rounded-[var(--radius-card)] shadow-sm border border-[var(--border-subtle)] p-6">
                    <h3 className="text-lg font-bold text-[var(--text-primary)] mb-3">
                        2. Watch a Folder
                    </h3>
                    <p className="text-[var(--text-secondary)] text-sm mb-4">
                        Start watching a directory for new audio files.
                    </p>
                    <div className="bg-[var(--bg-card)] rounded p-3 font-mono text-sm text-[var(--text-primary)] border border-[var(--border-subtle)]">
                        scriberr watch ~/Recordings
                    </div>
                </div>

                <div className="bg-[var(--bg-main)]/50 rounded-[var(--radius-card)] shadow-sm border border-[var(--border-subtle)] p-6">
                    <h3 className="text-lg font-bold text-[var(--text-primary)] mb-3">
                        3. Run as Service
                    </h3>
                    <p className="text-[var(--text-secondary)] text-sm mb-4">
                        Install as a background service to keep watching after restart.
                    </p>
                    <div className="bg-[var(--bg-card)] rounded p-3 font-mono text-sm text-[var(--text-primary)] border border-[var(--border-subtle)]">
                        sudo scriberr install ~/Recordings<br />
                        sudo scriberr start
                    </div>
                </div>
            </div>
        </div>
    )
}
