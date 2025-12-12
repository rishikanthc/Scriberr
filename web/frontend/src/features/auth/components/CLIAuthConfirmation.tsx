import { useState, useEffect } from "react";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Layout } from "@/components/Layout";

export function CLIAuthConfirmation() {
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const { getAuthHeaders } = useAuth()
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [user, setUser] = useState<{ id: number; username: string } | null>(null)
    const [processing, setProcessing] = useState(false)

    const callbackUrl = searchParams.get('callback_url')
    const deviceName = searchParams.get('device_name') || 'CLI Device'

    useEffect(() => {
        const checkSession = async () => {
            try {
                const res = await fetch('/api/v1/auth/cli/authorize', {
                    headers: getAuthHeaders(),
                })
                if (res.ok) {
                    const data = await res.json()
                    setUser(data.user)
                } else {
                    setError('You must be logged in to authorize the CLI.')
                }
            } catch (err) {
                setError('Failed to verify session.')
            } finally {
                setLoading(false)
            }
        }

        if (!callbackUrl) {
            setError('Invalid request: Missing callback URL.')
            setLoading(false)
            return
        }

        checkSession()
    }, [callbackUrl, getAuthHeaders])

    const handleApprove = async () => {
        setProcessing(true)
        try {
            const res = await fetch('/api/v1/auth/cli/authorize', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    ...getAuthHeaders(),
                },
                body: JSON.stringify({
                    callback_url: callbackUrl,
                    device_name: deviceName,
                }),
            })

            if (res.ok) {
                const data = await res.json()
                // Redirect to the CLI callback URL
                window.location.href = data.redirect_url
            } else {
                setError('Failed to authorize CLI.')
                setProcessing(false)
            }
        } catch (err) {
            setError('An error occurred.')
            setProcessing(false)
        }
    }

    const handleDeny = () => {
        navigate("/")
    }

    if (loading) {
        return (
            <Layout>
                <div className="flex items-center justify-center min-h-screen">
                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
                </div>
            </Layout>
        )
    }

    if (error) {
        return (
            <Layout>
                <div className="flex flex-col items-center justify-center min-h-[60vh] p-6">
                    <div className="bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 p-4 rounded-lg mb-4">
                        {error}
                    </div>
                    <button
                        onClick={() => navigate("/")}
                        className="px-4 py-2 bg-carbon-200 dark:bg-carbon-700 rounded hover:bg-carbon-300 dark:hover:bg-carbon-600"
                    >
                        Go Home
                    </button>
                </div>
            </Layout>
        )
    }

    return (
        <Layout>
            <div className="flex flex-col items-center justify-center min-h-[60vh] p-6">
                <div className="bg-white dark:bg-carbon-800 shadow-lg rounded-xl p-8 max-w-md w-full text-center">
                    <div className="w-16 h-16 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center mx-auto mb-6">
                        <svg className="w-8 h-8 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 18h.01M8 21h8a2 2 0 002-2V5a2 2 0 00-2-2H8a2 2 0 00-2 2v14a2 2 0 002 2z" />
                        </svg>
                    </div>

                    <h1 className="text-2xl font-bold text-carbon-900 dark:text-white mb-2">
                        Authorize CLI Device?
                    </h1>

                    <p className="text-carbon-600 dark:text-carbon-300 mb-6">
                        <span className="font-bold">{deviceName}</span> wants to access your account <span className="font-bold">{user?.username}</span>.
                    </p>

                    <div className="flex flex-col gap-3">
                        <button
                            onClick={handleApprove}
                            disabled={processing}
                            className="w-full py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            {processing ? 'Authorizing...' : 'Approve'}
                        </button>
                        <button
                            onClick={handleDeny}
                            disabled={processing}
                            className="w-full py-2.5 bg-carbon-100 dark:bg-carbon-700 hover:bg-carbon-200 dark:hover:bg-carbon-600 text-carbon-700 dark:text-carbon-200 rounded-lg font-medium transition-colors"
                        >
                            Deny
                        </button>
                    </div>
                </div>
            </div>
        </Layout>
    )
}
