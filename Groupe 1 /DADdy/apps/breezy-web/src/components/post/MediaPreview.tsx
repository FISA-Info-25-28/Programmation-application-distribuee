import { Loader2, AlertTriangle, X } from 'lucide-react'

export interface MediaItem {
  key: string
  previewUrl: string
  isVideo: boolean
  isAudio: boolean
  id?: number
  uploading: boolean
  error: boolean
  errorMessage?: string
  unsupported: boolean
}

export function MediaPreviewGrid({ items, onRemove }: { items: MediaItem[]; onRemove: (key: string) => void }) {
  const count = items.length
  const gridClass = count === 1 ? 'grid-cols-1' : 'grid-cols-2'

  return (
    <div className={`grid gap-0.5 mt-2 rounded-xl overflow-hidden ${gridClass}`}>
      {items.map((item, i) => (
        <div
          key={item.key}
          className={`relative bg-accent overflow-hidden ${count === 3 && i === 0 ? 'row-span-2' : ''}`}
          style={{ aspectRatio: count === 1 ? '16/9' : '1/1' }}
        >
          {item.isAudio ? (
            <div className="flex w-full h-full items-center justify-center bg-accent px-3">
              <audio src={item.previewUrl} controls className="w-full" />
            </div>
          ) : item.isVideo ? (
            <video src={item.previewUrl} className="w-full h-full object-cover" muted />
          ) : (
            <img src={item.previewUrl} alt="" className="w-full h-full object-cover" />
          )}

          {item.uploading && (
            <div className="absolute inset-0 bg-background/60 flex items-center justify-center">
              <Loader2 size={20} className="animate-spin text-primary" />
            </div>
          )}

          {item.unsupported && !item.uploading && !item.error && (
            <div className="absolute inset-0 bg-background/60 flex flex-col items-center justify-center gap-1">
              <AlertTriangle size={16} className="text-muted-foreground" />
              <span className="text-xs text-muted-foreground text-center px-2">Aperçu indisponible</span>
            </div>
          )}

          {item.error && (
            <div
              className="absolute inset-0 bg-destructive/20 flex items-center justify-center px-2"
              title={item.errorMessage}
            >
              <span className="text-xs text-destructive font-medium text-center">
                {item.errorMessage ?? 'Erreur'}
              </span>
            </div>
          )}

          <button
            type="button"
            onClick={() => onRemove(item.key)}
            className="absolute top-1 right-1 p-0.5 rounded-full bg-background/80 hover:bg-background text-foreground transition-colors"
          >
            <X size={12} />
          </button>
        </div>
      ))}
    </div>
  )
}
