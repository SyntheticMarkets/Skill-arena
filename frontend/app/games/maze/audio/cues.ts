export type MazeSoundCue = 'accepted' | 'blocked' | 'complete'

export function playMazeCue(cue: MazeSoundCue, enabled: boolean) {
  if (!enabled || typeof window === 'undefined') return
  const AudioContextClass = window.AudioContext
  if (!AudioContextClass) return
  const context = new AudioContextClass()
  const oscillator = context.createOscillator()
  const gain = context.createGain()
  const now = context.currentTime
  const frequency = cue === 'accepted' ? 520 : cue === 'blocked' ? 170 : 660
  oscillator.type = cue === 'blocked' ? 'square' : 'sine'
  oscillator.frequency.setValueAtTime(frequency, now)
  if (cue === 'complete') oscillator.frequency.exponentialRampToValueAtTime(980, now + 0.18)
  gain.gain.setValueAtTime(0.0001, now)
  gain.gain.exponentialRampToValueAtTime(0.055, now + 0.015)
  gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.22)
  oscillator.connect(gain)
  gain.connect(context.destination)
  oscillator.start(now)
  oscillator.stop(now + 0.24)
  oscillator.addEventListener('ended', () => void context.close())
}

