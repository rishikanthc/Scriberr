import { useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { ChevronDown, Upload, Mic, Settings } from 'lucide-react'
import { ScriberrLogo } from './ScriberrLogo'
import { ThemeSwitcher } from './ThemeSwitcher'
import { AudioRecorder } from './AudioRecorder'
import { useRouter } from '../contexts/RouterContext'

interface HeaderProps {
  onFileSelect: (file: File) => void
}

export function Header({ onFileSelect }: HeaderProps) {
  const { navigate } = useRouter()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [isRecorderOpen, setIsRecorderOpen] = useState(false)

  const handleUploadClick = () => {
    fileInputRef.current?.click()
  }

  const handleRecordClick = () => {
    setIsRecorderOpen(true)
  }

  const handleSettingsClick = () => {
    navigate({ path: 'settings' })
  }

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (file && file.type.startsWith('audio/')) {
      onFileSelect(file)
      // Reset the input so the same file can be selected again
      event.target.value = ''
    }
  }

  const handleRecordingComplete = async (blob: Blob, title: string) => {
    // Convert blob to file and use existing upload logic
    const file = new File([blob], `${title}.webm`, { type: blob.type })
    onFileSelect(file)
  }

  return (
    <header className="bg-white dark:bg-gray-800 rounded-xl p-6 mb-6">
      <div className="flex items-center justify-between">
        {/* Left side - Logo */}
        <ScriberrLogo />
        
        {/* Right side - Settings, Theme Switcher and Add Audio Dropdown */}
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleSettingsClick}
            className="h-10 w-10 p-0 rounded-xl hover:bg-gray-100 dark:hover:bg-gray-700 transition-all duration-200"
          >
            <Settings className="h-5 w-5 text-gray-600 dark:text-gray-400" />
          </Button>
          <ThemeSwitcher />
          
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button className="bg-blue-500 hover:bg-blue-600 text-white font-medium px-6 py-2 rounded-xl transition-all duration-300 hover:scale-[1.02] hover:shadow-lg hover:shadow-blue-500/20 gap-2">
                Add Audio
                <ChevronDown className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent 
              align="end" 
              className="w-48 bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700 shadow-lg"
            >
              <DropdownMenuItem
                onClick={handleUploadClick}
                className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300"
              >
                <Upload className="h-4 w-4 text-blue-500" />
                <div>
                  <div className="font-medium">Upload File</div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">Choose audio from device</div>
                </div>
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={handleRecordClick}
                className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300"
              >
                <Mic className="h-4 w-4 text-red-500" />
                <div>
                  <div className="font-medium">Record Audio</div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">Record using microphone</div>
                </div>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          
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

      {/* Audio Recorder Dialog */}
      <AudioRecorder
        isOpen={isRecorderOpen}
        onClose={() => setIsRecorderOpen(false)}
        onRecordingComplete={handleRecordingComplete}
      />
    </header>
  )
}