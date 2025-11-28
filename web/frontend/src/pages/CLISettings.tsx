import { useState } from 'react'
import { useRouter } from '../contexts/RouterContext'
import { Layout } from '../components/Layout'

export function CLISettings() {
    const { navigate } = useRouter()
    const [copied, setCopied] = useState(false)

    const installCommand = 'curl -sL https://scriberr.app/install.sh | bash'

    const copyCommand = () => {
        navigator.clipboard.writeText(installCommand)
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
    }

    return (
        <Layout>
            <div className="max-w-4xl mx-auto p-6">
                <div className="flex items-center justify-between mb-8">
                    <h1 className="text-2xl font-bold text-gray-900 dark:text-white">CLI Watcher</h1>
                    <button
                        onClick={() => navigate({ path: 'settings' })}
                        className="text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
                    >
                        Back to Settings
                    </button>
                </div>

                <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6 mb-6">
                    <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Installation</h2>
                    <p className="text-gray-600 dark:text-gray-300 mb-4">
                        The Scriberr CLI allows you to automatically upload audio files from a local folder.
                    </p>

                    <div className="bg-gray-900 rounded-lg p-4 relative group">
                        <code className="text-green-400 font-mono text-sm">{installCommand}</code>
                        <button
                            onClick={copyCommand}
                            className="absolute right-2 top-2 p-2 rounded bg-gray-700 hover:bg-gray-600 text-white opacity-0 group-hover:opacity-100 transition-opacity"
                            title="Copy to clipboard"
                        >
                            {copied ? (
                                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                                </svg>
                            ) : (
                                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
                                </svg>
                            )}
                        </button>
                    </div>
                </div>

                <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
                    <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Setup Instructions</h2>
                    <ol className="list-decimal list-inside space-y-3 text-gray-600 dark:text-gray-300">
                        <li>Run the installation command above.</li>
                        <li>
                            Authenticate the CLI with your account:
                            <pre className="mt-2 bg-gray-100 dark:bg-gray-900 p-2 rounded text-sm font-mono inline-block">
                                scriberr login
                            </pre>
                        </li>
                        <li>
                            Start watching a folder:
                            <pre className="mt-2 bg-gray-100 dark:bg-gray-900 p-2 rounded text-sm font-mono inline-block">
                                scriberr watch ~/Recordings
                            </pre>
                        </li>
                        <li>
                            Or run as a background service:
                            <pre className="mt-2 bg-gray-100 dark:bg-gray-900 p-2 rounded text-sm font-mono inline-block">
                                scriberr install
                                scriberr start
                            </pre>
                        </li>
                    </ol>
                </div>
            </div>
        </Layout>
    )
}
