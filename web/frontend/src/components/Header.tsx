import { useRef } from 'react'
import { Button } from '@/components/ui/button'
import { ScriberrLogo } from './ScriberrLogo'
import { ThemeSwitcher } from './ThemeSwitcher'

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
    <header className="bg-white dark:bg-gray-800 rounded-xl p-6 mb-6">
      <div className="flex items-center justify-between">
        {/* Left side - Logo */}
        <ScriberrLogo />
        
        {/* Right side - Theme Switcher and Add Audio Button */}
        <div className="flex items-center gap-4">
          <ThemeSwitcher />
          <Button
            onClick={handleAddAudioClick}
            className="bg-blue-500 hover:bg-blue-600 text-white font-medium px-6 py-2 rounded-xl transition-all duration-300 hover:scale-[1.02] hover:shadow-lg hover:shadow-blue-500/20"
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