import { useEffect, useMemo, useState, useRef } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

type AdminUser = {
  id: number
  username: string
  is_admin: boolean
  created_at: string
  updated_at: string
}

export function Admin() {
  const { getAuthHeaders, token } = useAuth()

  const [users, setUsers] = useState<AdminUser[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Create user form state
  const [newUsername, setNewUsername] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [newConfirm, setNewConfirm] = useState('')
  const [newIsAdmin, setNewIsAdmin] = useState(false)
  const [creating, setCreating] = useState(false)

  // Simple password reset state per user id
  const [pwdTarget, setPwdTarget] = useState<number | null>(null)
  const [pwd1, setPwd1] = useState('')
  const [pwd2, setPwd2] = useState('')

  // Logo/branding state
  const [logoUploading, setLogoUploading] = useState(false)
  const [logoVersion, setLogoVersion] = useState(0)
  const [logoError, setLogoError] = useState<string | null>(null)
  const logoInputRef = useRef<HTMLInputElement | null>(null)

  const canCreate = useMemo(
    () =>
      newUsername.trim().length >= 3 &&
      newPassword.length >= 6 &&
      newPassword === newConfirm,
    [newUsername, newPassword, newConfirm]
  )

  const fetchUsers = async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/v1/admin/users/', {
        headers: { ...getAuthHeaders() }
      })
      if (!res.ok) {
        throw new Error('Failed to load users')
      }
      const data = await res.json()
      setUsers(data as AdminUser[])
    } catch (e: any) {
      setError(e?.message || 'Failed to load users')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (token) {
      fetchUsers()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token])

  const createUser = async () => {
    if (!canCreate) return
    setCreating(true)
    setError(null)
    try {
      const res = await fetch('/api/v1/admin/users/', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        },
        body: JSON.stringify({
          username: newUsername.trim(),
          password: newPassword,
          confirmPassword: newConfirm,
          is_admin: newIsAdmin
        })
      })
      const data = await res.json()
      if (!res.ok) {
        throw new Error(data?.error || 'Failed to create user')
      }
      setNewUsername('')
      setNewPassword('')
      setNewConfirm('')
      setNewIsAdmin(false)
      await fetchUsers()
    } catch (e: any) {
      setError(e?.message || 'Failed to create user')
    } finally {
      setCreating(false)
    }
  }

  const toggleAdmin = async (u: AdminUser) => {
    try {
      const res = await fetch(`/api/v1/admin/users/${u.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        },
        body: JSON.stringify({ is_admin: !u.is_admin })
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) {
        throw new Error((data as any)?.error || 'Failed to update user')
      }
      await fetchUsers()
    } catch (e: any) {
      setError(e?.message || 'Failed to update user')
    }
  }

  const deleteUser = async (u: AdminUser) => {
    if (!confirm(`Delete user ${u.username}?`)) return
    try {
      const res = await fetch(`/api/v1/admin/users/${u.id}`, {
        method: 'DELETE',
        headers: { ...getAuthHeaders() }
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) {
        throw new Error((data as any)?.error || 'Failed to delete user')
      }
      await fetchUsers()
    } catch (e: any) {
      setError(e?.message || 'Failed to delete user')
    }
  }

  const resetPassword = async (u: AdminUser) => {
    if (!pwd1 || pwd1.length < 6 || pwd1 !== pwd2) {
      setError('Passwords must match and be at least 6 chars')
      return
    }
    try {
      const res = await fetch(`/api/v1/admin/users/${u.id}/password`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        },
        body: JSON.stringify({
          newPassword: pwd1,
          confirmPassword: pwd2
        })
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) {
        throw new Error((data as any)?.error || 'Failed to update password')
      }
      setPwdTarget(null)
      setPwd1('')
      setPwd2('')
    } catch (e: any) {
      setError(e?.message || 'Failed to update password')
    }
  }

  const handleLogoChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return

    setLogoError(null)

    if (!file.type.startsWith('image/')) {
      setLogoError('Please select an image file (PNG).')
      event.target.value = ''
      return
    }

    const formData = new FormData()
    formData.append('logo', file)

    setLogoUploading(true)
    try {
      const res = await fetch('/api/v1/admin/logo', {
        method: 'POST',
        headers: {
          ...getAuthHeaders()
          // Do not set Content-Type, browser will set multipart boundary
        },
        body: formData
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) {
        throw new Error((data as any)?.error || 'Failed to upload logo')
      }
      setLogoVersion(v => v + 1)
    } catch (e: any) {
      setLogoError(e?.message || 'Failed to upload logo')
    } finally {
      setLogoUploading(false)
      event.target.value = ''
    }
  }

  const handleLogoReset = async () => {
    setLogoError(null)
    setLogoUploading(true)
    try {
      const res = await fetch('/api/v1/admin/logo', {
        method: 'DELETE',
        headers: {
          ...getAuthHeaders()
        }
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) {
        throw new Error((data as any)?.error || 'Failed to reset logo')
      }
      setLogoVersion(v => v + 1)
    } catch (e: any) {
      setLogoError(e?.message || 'Failed to reset logo')
    } finally {
      setLogoUploading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="max-w-5xl mx-auto p-4 sm:p-6">
        <Card className="mb-6 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
          <CardHeader>
            <CardTitle className="text-gray-900 dark:text-gray-100">User Management</CardTitle>
            <CardDescription className="text-gray-600 dark:text-gray-400">
              Add, promote, or remove users for this Scriberr instance
            </CardDescription>
          </CardHeader>
          <CardContent>
            {error && (
              <div className="mb-4 text-sm text-red-600 dark:text-red-400">{error}</div>
            )}
            <div className="grid grid-cols-1 md:grid-cols-5 gap-3 items-end">
              <div className="md:col-span-2">
                <Label
                  htmlFor="newUsername"
                  className="text-gray-700 dark:text-gray-300"
                >
                  Username
                </Label>
                <Input
                  id="newUsername"
                  value={newUsername}
                  onChange={e => setNewUsername(e.target.value)}
                  placeholder="e.g. alice"
                  className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100"
                />
              </div>
              <div>
                <Label
                  htmlFor="newPassword"
                  className="text-gray-700 dark:text-gray-300"
                >
                  Password
                </Label>
                <Input
                  id="newPassword"
                  type="password"
                  value={newPassword}
                  onChange={e => setNewPassword(e.target.value)}
                  className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100"
                />
              </div>
              <div>
                <Label
                  htmlFor="newConfirm"
                  className="text-gray-700 dark:text-gray-300"
                >
                  Confirm
                </Label>
                <Input
                  id="newConfirm"
                  type="password"
                  value={newConfirm}
                  onChange={e => setNewConfirm(e.target.value)}
                  className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100"
                />
              </div>
              <div className="flex items-center gap-3">
                <label className="flex items-center gap-2 text-gray-700 dark:text-gray-300">
                  <input
                    type="checkbox"
                    checked={newIsAdmin}
                    onChange={e => setNewIsAdmin(e.target.checked)}
                  />
                  Admin
                </label>
                <Button
                  disabled={!canCreate || creating}
                  onClick={createUser}
                  className="bg-blue-600 hover:bg-blue-700 text-white"
                >
                  Create
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
          <CardHeader>
            <CardTitle className="text-gray-900 dark:text-gray-100">Users</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="text-gray-600 dark:text-gray-300">Loading users…</div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Username</TableHead>
                    <TableHead>Admin</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {users.map(u => (
                    <TableRow key={u.id}>
                      <TableCell className="font-medium">{u.username}</TableCell>
                      <TableCell>{u.is_admin ? 'Yes' : 'No'}</TableCell>
                      <TableCell className="space-x-2">
                        <Button
                          variant="outline"
                          onClick={() => toggleAdmin(u)}
                        >
                          {u.is_admin ? 'Remove Admin' : 'Make Admin'}
                        </Button>
                        {pwdTarget === u.id ? (
                          <span className="inline-flex items-center gap-2">
                            <Input
                              type="password"
                              placeholder="New password"
                              value={pwd1}
                              onChange={e => setPwd1(e.target.value)}
                              className="w-40 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100"
                            />
                            <Input
                              type="password"
                              placeholder="Confirm"
                              value={pwd2}
                              onChange={e => setPwd2(e.target.value)}
                              className="w-40 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100"
                            />
                            <Button size="sm" onClick={() => resetPassword(u)}>
                              Save
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => {
                                setPwdTarget(null)
                                setPwd1('')
                                setPwd2('')
                              }}
                            >
                              Cancel
                            </Button>
                          </span>
                        ) : (
                          <Button
                            variant="outline"
                            onClick={() => setPwdTarget(u.id)}
                          >
                            Reset Password
                          </Button>
                        )}
                        <Button
                          variant="destructive"
                          onClick={() => deleteUser(u)}
                        >
                          Delete
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>

        <Card className="mt-6 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
          <CardHeader>
            <CardTitle className="text-gray-900 dark:text-gray-100">Branding</CardTitle>
            <CardDescription className="text-gray-600 dark:text-gray-400">
              Upload a custom logo for this Scriberr instance.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {logoError && (
              <div className="mb-4 text-sm text-red-600 dark:text-red-400">{logoError}</div>
            )}
            <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4">
              <div>
                <Label className="text-gray-700 dark:text-gray-300 mb-1 block">
                  Current logo
                </Label>
                <div className="p-2 rounded-md border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 inline-flex">
                  <img
                    src={`/scriberr-logo.png?v=${logoVersion}`}
                    alt="Current logo"
                    className="h-12 w-auto"
                  />
                </div>
              </div>
              <div className="flex flex-col gap-2">
                <Label
                  htmlFor="logoUpload"
                  className="text-gray-700 dark:text-gray-300"
                >
                  Change logo (PNG)
                </Label>
                <input
                  id="logoUpload"
                  type="file"
                  accept="image/png"
                  onChange={handleLogoChange}
                  disabled={logoUploading}
                  ref={logoInputRef}
                  className="hidden"
                />
                <div className="flex gap-2">
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => logoInputRef.current?.click()}
                    disabled={logoUploading}
                    className="border-blue-200 text-blue-700 hover:bg-blue-50 hover:text-blue-800"
                  >
                    Change logo
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={handleLogoReset}
                    disabled={logoUploading}
                  >
                    Reset to default
                  </Button>
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Changes are applied immediately after upload.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

export default Admin
