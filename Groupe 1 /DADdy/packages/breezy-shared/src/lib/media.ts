const HEIC_MIME_TYPES = new Set(['image/heic', 'image/heif', 'image/heic-sequence', 'image/heif-sequence'])

function isHeicByExtension(name: string): boolean {
  return /\.(heic|heif)$/i.test(name)
}

function canBrowserPlayVideo(mimeType: string): boolean {
  if (!mimeType) return true
  const video = document.createElement('video')
  return video.canPlayType(mimeType) !== ''
}

function canBrowserPlayAudio(mimeType: string): boolean {
  if (!mimeType) return true
  const audio = document.createElement('audio')
  return audio.canPlayType(mimeType) !== ''
}


function canBrowserShowImage(mimeType: string): boolean {
  if (!mimeType) return true
  const img = document.createElement('img')
  // Check via canvas-based test or known list
  const supported = new Set([
    'image/jpeg', 'image/jpg', 'image/png', 'image/gif',
    'image/webp', 'image/avif', 'image/bmp', 'image/svg+xml',
    'image/x-icon', 'image/ico',
  ])
  return supported.has(mimeType.toLowerCase()) ||
    (typeof img.decode === 'function' && mimeType.startsWith('image/'))
}

export interface ProcessedFile {
  file: File
  previewUrl: string
  isVideo: boolean
  isAudio: boolean
  /** True when the browser can't preview the format (upload still proceeds) */
  unsupported: boolean
}

export async function processMediaFile(file: File): Promise<ProcessedFile> {
  const type = file.type.toLowerCase()
  const isVideo = type.startsWith('video/')

  const isAudio = type.startsWith('audio/')


  // HEIC/HEIF: convert to JPEG before preview and upload
  if (HEIC_MIME_TYPES.has(type) || (!type && isHeicByExtension(file.name))) {
    try {
      const { default: heic2any } = await import('heic2any')
      const result = await heic2any({ blob: file, toType: 'image/jpeg', quality: 0.92 })
      const blob = Array.isArray(result) ? result[0] : result
      const converted = new File(
        [blob],
        file.name.replace(/\.(heic|heif)$/i, '.jpg'),
        { type: 'image/jpeg' },
      )
      return { file: converted, previewUrl: URL.createObjectURL(converted), isVideo: false, isAudio: false, unsupported: false }
    } catch {
      return { file, previewUrl: '', isVideo: false, isAudio: false, unsupported: true }
    }
  }

  // Video: upload always, but flag if browser can't play
  if (isVideo) {
    const playable = canBrowserPlayVideo(type)
    return { file, previewUrl: URL.createObjectURL(file), isVideo: true, isAudio: false, unsupported: !playable }
  }

  // Audio: upload always, but flag if browser can't play
  if (isAudio) {
    const playable = canBrowserPlayAudio(type)
    return { file, previewUrl: URL.createObjectURL(file), isVideo: false, isAudio: true, unsupported: !playable }
  }

  // Other images: upload, flag if browser likely can't show
  const viewable = canBrowserShowImage(type)
  return { file, previewUrl: URL.createObjectURL(file), isVideo: false, isAudio: false, unsupported: !viewable }
}
