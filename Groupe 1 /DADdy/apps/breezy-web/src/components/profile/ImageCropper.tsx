import { useState, useRef, useEffect, useCallback } from 'react'
import { Loader2, ZoomIn } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface ImageCropperProps {
  /** Fichier source choisi par l'utilisateur. */
  file: File
  /** Ratio largeur/hauteur de la zone de recadrage (1 = carré, 3 = bannière). */
  aspect: number
  /** Affiche un masque circulaire (avatar). */
  round?: boolean
  /** Largeur d'export en pixels ; la hauteur est déduite de `aspect`. */
  outputWidth: number
  title: string
  zoomLabel: string
  applyLabel: string
  cancelLabel: string
  hint: string
  onCancel: () => void
  onConfirm: (file: File) => void
}

const MIN_ZOOM = 1
const MAX_ZOOM = 4

// ImageCropper affiche une fenêtre de recadrage : l'image est déplaçable (drag)
// et zoomable (molette ou curseur). Au clic sur « Appliquer », la portion visible
// est redessinée sur un canvas à la taille d'export demandée puis renvoyée sous
// forme de File JPEG, prête à être uploadée par l'appelant.
export function ImageCropper({
  file,
  aspect,
  round,
  outputWidth,
  title,
  zoomLabel,
  applyLabel,
  cancelLabel,
  hint,
  onCancel,
  onConfirm,
}: ImageCropperProps) {
  const viewW = round ? 288 : 432
  const viewH = Math.round(viewW / aspect)

  const [img, setImg] = useState<HTMLImageElement | null>(null)
  const [zoom, setZoom] = useState(MIN_ZOOM)
  const [offset, setOffset] = useState({ x: 0, y: 0 })
  const [busy, setBusy] = useState(false)

  // Échelle de base pour que l'image couvre toujours la zone (object-fit: cover).
  const [cover, setCover] = useState(1)
  const dragRef = useRef<{ px: number; py: number; ox: number; oy: number } | null>(null)

  useEffect(() => {
    const url = URL.createObjectURL(file)
    const image = new Image()
    image.onload = () => {
      const coverScale = Math.max(viewW / image.naturalWidth, viewH / image.naturalHeight)
      setCover(coverScale)
      const dispW = image.naturalWidth * coverScale
      const dispH = image.naturalHeight * coverScale
      setOffset({ x: (viewW - dispW) / 2, y: (viewH - dispH) / 2 })
      setZoom(MIN_ZOOM)
      setImg(image)
    }
    image.src = url
    return () => URL.revokeObjectURL(url)
  }, [file, viewW, viewH])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onCancel])

  // clamp empêche tout bord de l'image de rentrer dans la zone (toujours couverte).
  const clamp = useCallback(
    (x: number, y: number, eff: number) => {
      if (!img) return { x, y }
      const dispW = img.naturalWidth * eff
      const dispH = img.naturalHeight * eff
      return {
        x: Math.min(0, Math.max(viewW - dispW, x)),
        y: Math.min(0, Math.max(viewH - dispH, y)),
      }
    },
    [img, viewW, viewH],
  )

  const effScale = cover * zoom

  // applyZoom modifie le zoom en gardant fixe le point au centre de la zone.
  const applyZoom = (next: number) => {
    const nz = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, next))
    setOffset((prev) => {
      const oldEff = cover * zoom
      const newEff = cover * nz
      const cx = (viewW / 2 - prev.x) / oldEff
      const cy = (viewH / 2 - prev.y) / oldEff
      return clamp(viewW / 2 - cx * newEff, viewH / 2 - cy * newEff, newEff)
    })
    setZoom(nz)
  }

  const onPointerDown = (e: React.PointerEvent) => {
    e.currentTarget.setPointerCapture(e.pointerId)
    dragRef.current = { px: e.clientX, py: e.clientY, ox: offset.x, oy: offset.y }
  }
  const onPointerMove = (e: React.PointerEvent) => {
    const d = dragRef.current
    if (!d) return
    setOffset(clamp(d.ox + (e.clientX - d.px), d.oy + (e.clientY - d.py), effScale))
  }
  const onPointerUp = (e: React.PointerEvent) => {
    dragRef.current = null
    try {
      e.currentTarget.releasePointerCapture(e.pointerId)
    } catch {
      // pointer déjà relâché : sans effet
    }
  }
  const onWheel = (e: React.WheelEvent) => {
    applyZoom(zoom - e.deltaY * 0.002)
  }

  const handleConfirm = async () => {
    if (!img) return
    setBusy(true)
    try {
      const eff = effScale
      const outH = Math.round(outputWidth / aspect)
      const canvas = document.createElement('canvas')
      canvas.width = outputWidth
      canvas.height = outH
      const ctx = canvas.getContext('2d')
      if (!ctx) {
        setBusy(false)
        return
      }
      ctx.imageSmoothingQuality = 'high'
      ctx.drawImage(img, -offset.x / eff, -offset.y / eff, viewW / eff, viewH / eff, 0, 0, outputWidth, outH)
      const blob = await new Promise<Blob | null>((res) => canvas.toBlob(res, 'image/jpeg', 0.9))
      if (!blob) {
        setBusy(false)
        return
      }
      const base = file.name.replace(/\.[^.]+$/, '') || 'image'
      onConfirm(new File([blob], `${base}.jpg`, { type: 'image/jpeg' }))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center" aria-modal="true" role="dialog">
      <div className="absolute inset-0 bg-background/40 backdrop-blur-[2px]" onClick={onCancel} />

      <div className="relative z-10 w-full max-w-md mx-4 rounded-2xl border border-border bg-card shadow-xl p-4">
        <p className="text-sm font-semibold text-foreground mb-3">{title}</p>

        <div
          className="relative mx-auto overflow-hidden rounded-lg bg-black/80 touch-none select-none cursor-grab active:cursor-grabbing"
          style={{ width: viewW, height: viewH }}
          onPointerDown={onPointerDown}
          onPointerMove={onPointerMove}
          onPointerUp={onPointerUp}
          onPointerLeave={onPointerUp}
          onWheel={onWheel}
        >
          {img ? (
            <img
              src={img.src}
              alt=""
              draggable={false}
              className="absolute left-0 top-0 max-w-none pointer-events-none"
              style={{
                width: img.naturalWidth * effScale,
                height: img.naturalHeight * effScale,
                transform: `translate(${offset.x}px, ${offset.y}px)`,
              }}
            />
          ) : (
            <div className="absolute inset-0 flex items-center justify-center">
              <Loader2 size={24} className="animate-spin text-white" />
            </div>
          )}
          {round ? (
            <div className="absolute inset-0 rounded-full ring-[9999px] ring-black/55 pointer-events-none" />
          ) : (
            // Cadre matérialisant exactement la zone qui sera affichée.
            <div className="absolute inset-0 rounded-lg ring-2 ring-inset ring-white/80 pointer-events-none" />
          )}
        </div>

        <p className="text-xs text-muted-foreground text-center mt-2">{hint}</p>

        <div className="flex items-center gap-3 mt-3">
          <ZoomIn size={15} className="text-muted-foreground shrink-0" />
          <input
            type="range"
            aria-label={zoomLabel}
            min={MIN_ZOOM}
            max={MAX_ZOOM}
            step={0.01}
            value={zoom}
            onChange={(e) => applyZoom(Number(e.target.value))}
            className="flex-1 accent-primary"
          />
        </div>

        <div className="flex justify-end gap-2 mt-4">
          <Button variant="ghost" size="sm" onClick={onCancel} disabled={busy}>
            {cancelLabel}
          </Button>
          <Button size="sm" onClick={handleConfirm} disabled={!img || busy} className="gap-1.5">
            {busy && <Loader2 size={14} className="animate-spin" />}
            {applyLabel}
          </Button>
        </div>
      </div>
    </div>
  )
}
