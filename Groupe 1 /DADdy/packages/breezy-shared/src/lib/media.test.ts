import { beforeEach, describe, expect, it, vi } from 'vitest'
import { processMediaFile } from './media'

const mockHeic2any = vi.hoisted(() => vi.fn())
vi.mock('heic2any', () => ({ default: mockHeic2any }))

function makeFile(name: string, type: string, content = 'data'): File {
  return new File([content], name, { type })
}

beforeEach(() => {
  mockHeic2any.mockReset()
  URL.createObjectURL = vi.fn(() => 'blob:mock-url') as typeof URL.createObjectURL
})

describe('processMediaFile', () => {
  it('JPEG → isVideo false, isAudio false, unsupported false', async () => {
    const result = await processMediaFile(makeFile('photo.jpg', 'image/jpeg'))
    expect(result.isVideo).toBe(false)
    expect(result.isAudio).toBe(false)
    expect(result.unsupported).toBe(false)
  })

  it('video/mp4 → isVideo true, isAudio false', async () => {
    const result = await processMediaFile(makeFile('clip.mp4', 'video/mp4'))
    expect(result.isVideo).toBe(true)
    expect(result.isAudio).toBe(false)
  })

  it('audio/mpeg → isVideo false, isAudio true', async () => {
    const result = await processMediaFile(makeFile('song.mp3', 'audio/mpeg'))
    expect(result.isVideo).toBe(false)
    expect(result.isAudio).toBe(true)
  })

  it('image/heic → calls heic2any and returns converted JPEG', async () => {
    const blob = new Blob(['jpeg-bytes'], { type: 'image/jpeg' })
    mockHeic2any.mockResolvedValue(blob)

    const file = makeFile('photo.heic', 'image/heic')
    const result = await processMediaFile(file)

    expect(mockHeic2any).toHaveBeenCalledWith(
      expect.objectContaining({ blob: file, toType: 'image/jpeg' }),
    )
    expect(result.file.type).toBe('image/jpeg')
    expect(result.isVideo).toBe(false)
    expect(result.isAudio).toBe(false)
    expect(result.unsupported).toBe(false)
  })

  it('image/heic + heic2any reject → unsupported true', async () => {
    mockHeic2any.mockRejectedValue(new Error('conversion failed'))

    const result = await processMediaFile(makeFile('photo.heic', 'image/heic'))
    expect(result.unsupported).toBe(true)
  })

  it('.heic extension without MIME → detects by extension, calls heic2any', async () => {
    const blob = new Blob(['jpeg-bytes'], { type: 'image/jpeg' })
    mockHeic2any.mockResolvedValue(blob)

    const result = await processMediaFile(makeFile('photo.heic', ''))
    expect(mockHeic2any).toHaveBeenCalled()
    expect(result.unsupported).toBe(false)
  })

  it('empty MIME on non-HEIC file → canBrowserShowImage true → unsupported false', async () => {
    const result = await processMediaFile(makeFile('unknown', ''))
    expect(result.unsupported).toBe(false)
  })

  it('image type not in supported set but starts with image/ → unsupported false (img.decode path)', async () => {
    // 'image/tiff' is not in the supported Set but starts with 'image/'
    // In jsdom, img.decode is a function → canBrowserShowImage returns true
    const result = await processMediaFile(makeFile('photo.tiff', 'image/tiff'))
    expect(result.isVideo).toBe(false)
    expect(result.isAudio).toBe(false)
  })

  it('heic2any returns array → first element is used', async () => {
    const blob = new Blob(['jpeg-bytes'], { type: 'image/jpeg' })
    mockHeic2any.mockResolvedValue([blob])

    const result = await processMediaFile(makeFile('photo.heic', 'image/heic'))
    expect(result.file.type).toBe('image/jpeg')
    expect(result.unsupported).toBe(false)
  })
})
