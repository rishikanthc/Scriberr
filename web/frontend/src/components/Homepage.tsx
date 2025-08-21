import { useState } from 'react'
import { Header } from './Header'
import { AudioFilesTable } from './AudioFilesTable'

export function Homepage() {
  const [refreshTrigger, setRefreshTrigger] = useState(0)

  const handleFileSelect = async (file: File) => {
    const formData = new FormData()
    formData.append('audio', file)
    formData.append('title', file.name.replace(/\.[^/.]+$/, ""))

    try {
      const response = await fetch('/api/v1/transcription/upload', {
        method: 'POST',
        headers: {
          'X-API-Key': 'dev-api-key-123'
        },
        body: formData
      })

      if (response.ok) {
        // Refresh the table to show the new file
        setRefreshTrigger(prev => prev + 1)
      } else {
        alert('Failed to upload file')
      }
    } catch (error) {
      alert('Error uploading file')
    }
  }

  return (
    <div className="min-h-screen bg-gray-900">
      <div className="mx-auto px-8 py-6" style={{ width: '90vw' }}>
        <Header onFileSelect={handleFileSelect} />
        <AudioFilesTable refreshTrigger={refreshTrigger} />
      </div>
    </div>
  )
}