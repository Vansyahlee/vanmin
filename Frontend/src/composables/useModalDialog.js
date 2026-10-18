import { nextTick, onBeforeUnmount, watch } from 'vue'

export const useModalDialog = (getSelector, closeModal) => {
  let previousFocus = null
  let previousOverflow = ''
  let previousPadding = ''
  let locked = false

  const unlock = () => {
    if (!locked) {
      return
    }

    document.body.style.overflow = previousOverflow
    document.body.style.paddingRight = previousPadding
    locked = false
    previousFocus?.focus?.()
  }

  watch(getSelector, async selector => {
    if (!selector) {
      unlock()
      return
    }

    if (!locked) {
      previousFocus = document.activeElement
      previousOverflow = document.body.style.overflow
      previousPadding = document.body.style.paddingRight
      const scrollbar = window.innerWidth - document.documentElement.clientWidth
      const padding = parseFloat(getComputedStyle(document.body).paddingRight) || 0
      document.body.style.overflow = 'hidden'
      document.body.style.paddingRight = `${padding + scrollbar}px`
      locked = true
    }

    await nextTick()

    if (getSelector() !== selector) {
      return
    }

    const modal = document.querySelector(selector)
    const first = modal?.querySelector('input[autofocus], input:not([type="hidden"]):not(:disabled), button:not(:disabled)')
    first?.focus()
  })

  const keydown = event => {
    const selector = getSelector()

    if (!selector) {
      return
    }

    if (event.key === 'Escape') {
      if (!document.querySelector('.fas-choice-menu')) {
        event.preventDefault()
        closeModal()
      }

      return
    }

    if (event.key !== 'Tab') {
      return
    }

    const modal = document.querySelector(selector)
    const focusable = Array.from(modal?.querySelectorAll('button:not(:disabled), input:not([type="hidden"]):not(:disabled), textarea:not(:disabled), select:not(:disabled), a[href], [tabindex="0"]') || [])
      .filter(element => element.getClientRects().length > 0)
    const first = focusable[0]
    const last = focusable.at(-1)

    if (event.shiftKey && (document.activeElement === first || !modal?.contains(document.activeElement))) {
      event.preventDefault()
      last?.focus()
    } else if (!event.shiftKey && (document.activeElement === last || !modal?.contains(document.activeElement))) {
      event.preventDefault()
      first?.focus()
    }
  }

  document.addEventListener('keydown', keydown, true)

  onBeforeUnmount(() => {
    document.removeEventListener('keydown', keydown, true)
    unlock()
  })
}
