import { useRef } from 'react'
import { Button } from '@/components/ui/button'
import { ScriberrLogo } from './ScriberrLogo'

interface HeaderProps {
  onFileSelect: (file: File) => void
}

export function Header({ onFileSelect }: HeaderProps) {
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleAddAudioClick = () => {
    fileInputRef.current?.click()
  }

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (file && file.type.startsWith('audio/')) {
      onFileSelect(file)
      // Reset the input so the same file can be selected again
      event.target.value = ''
    }
  }

  return (
    <header className="bg-gray-800 rounded-xl p-8 mb-8">
      <div className="flex items-center justify-between">
        {/* Left side - Logo */}
        <ScriberrLogo />
        
        {/* Right side - Add Audio Button */}
        <div className="flex items-center gap-4">
          <Button
            onClick={handleAddAudioClick}
            className="bg-neon-100 hover:bg-neon-200 text-gray-900 font-medium px-8 py-3 rounded-xl transition-all duration-300 hover:scale-[1.02] hover:shadow-lg hover:shadow-neon-100/20"
          >
            Add Audio
          </Button>
          
          {/* Hidden file input */}
          <input
            ref={fileInputRef}
            type="file"
            accept="audio/*"
            onChange={handleFileChange}
            className="hidden"
          />
        </div>
      </div>
    </header>
  )
}