<script>
import {
  defineComponent,
  h,
  nextTick,
  onBeforeUnmount,
  onMounted,
  onUpdated,
  ref,
  Teleport,
} from 'vue'

import FasIcon from './FasIcon.vue'

let selectSequence = 0

const textContent = value => {
  if (Array.isArray(value)) {
    return value.map(textContent).join('')
  }

  if (value && typeof value === 'object') {
    return textContent(value.children)
  }

  return value == null ? '' : String(value)
}

const readOptions = nodes => {
  const result = []

  for (const node of nodes || []) {
    if (node.type === 'option') {
      const label = textContent(node.children).replace(/\s+/g, ' ').trim()
      const value = node.props?.value ?? label

      result.push({
        value,
        label,
        disabled: node.props?.disabled != null && node.props.disabled !== false,
      })
    } else if (Array.isArray(node.children)) {
      result.push(...readOptions(node.children))
    }
  }

  return result
}

export default defineComponent({
  name: 'FasSelect',
  inheritAttrs: false,

  props: {
    modelValue: {
      type: [String, Number, Boolean],
      default: '',
    },
    disabled: Boolean,
    required: Boolean,
    name: String,
    placeholder: {
      type: String,
      default: 'Pilih opsi',
    },
  },

  emits: ['update:modelValue', 'change'],

  setup(props, { attrs, emit, slots }) {
    const open = ref(false)
    const root = ref(null)
    const trigger = ref(null)
    const menu = ref(null)
    const activeIndex = ref(0)
    const menuStyle = ref({})
    const listID = `fas-choice-${++selectSequence}`
    let options = []
    let searchText = ''
    let searchTimer = null

    const positionMenu = () => {
      if (!open.value || !trigger.value) {
        return
      }

      const rect = trigger.value.getBoundingClientRect()
      const width = Math.min(rect.width, window.innerWidth - 24)
      const below = window.innerHeight - rect.bottom - 16
      const above = rect.top - 16
      const upwards = below < 200 && above > below
      const height = Math.min(320, Math.max(80, upwards ? above : below))
      const nextStyle = {
        width: `${width}px`,
        left: `${Math.max(12, Math.min(rect.left, window.innerWidth - width - 12))}px`,
        maxHeight: `${height}px`,
        ...(upwards
          ? { bottom: `${window.innerHeight - rect.top + 6}px`, top: 'auto' }
          : { top: `${rect.bottom + 6}px`, bottom: 'auto' }),
      }

      if (JSON.stringify(menuStyle.value) !== JSON.stringify(nextStyle)) {
        menuStyle.value = nextStyle
      }
    }

    const show = () => {
      if (props.disabled || options.length === 0) {
        return
      }

      activeIndex.value = Math.max(0, options.findIndex(option => String(option.value) === String(props.modelValue)))
      open.value = true
      nextTick(positionMenu)
    }

    const close = () => {
      open.value = false
    }

    const choose = option => {
      if (option.disabled || props.disabled) {
        return
      }

      emit('update:modelValue', option.value)
      emit('change', { target: { value: option.value } })
      close()
      trigger.value?.focus()
    }

    const scrollActive = () => {
      nextTick(() => {
        menu.value?.querySelector(`[data-index="${activeIndex.value}"]`)?.scrollIntoView({ block: 'nearest' })
      })
    }

    const move = direction => {
      if (!open.value) {
        show()
        return
      }

      let next = activeIndex.value

      for (let count = 0; count < options.length; count += 1) {
        next = (next + direction + options.length) % options.length

        if (!options[next].disabled) {
          activeIndex.value = next
          scrollActive()
          break
        }
      }
    }

    const keydown = event => {
      if (['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(event.key)) {
        event.preventDefault()

        if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
          move(event.key === 'ArrowDown' ? 1 : -1)
        } else if (open.value && options[activeIndex.value]) {
          choose(options[activeIndex.value])
        } else {
          show()
        }
      } else if (event.key === 'Escape' || event.key === 'Tab') {
        close()
      } else if (open.value && (event.key === 'Home' || event.key === 'End')) {
        event.preventDefault()
        const enabled = options.map((option, index) => ({ ...option, index })).filter(option => !option.disabled)
        activeIndex.value = event.key === 'Home' ? enabled[0]?.index || 0 : enabled.at(-1)?.index || 0
        scrollActive()
      } else if (event.key.length === 1 && !event.ctrlKey && !event.metaKey && !event.altKey) {
        clearTimeout(searchTimer)
        searchText += event.key.toLocaleLowerCase('id-ID')
        const index = options.findIndex(option => !option.disabled && option.label.toLocaleLowerCase('id-ID').startsWith(searchText))

        if (index >= 0) {
          if (!open.value) {
            show()
          }

          activeIndex.value = index
          scrollActive()
        }

        searchTimer = setTimeout(() => {
          searchText = ''
        }, 700)
      }
    }

    const outside = event => {
      if (!root.value?.contains(event.target) && !menu.value?.contains(event.target)) {
        close()
      }
    }

    onMounted(() => {
      document.addEventListener('pointerdown', outside)
      window.addEventListener('resize', positionMenu)
      window.addEventListener('scroll', positionMenu, true)
    })

    onUpdated(positionMenu)

    onBeforeUnmount(() => {
      document.removeEventListener('pointerdown', outside)
      window.removeEventListener('resize', positionMenu)
      window.removeEventListener('scroll', positionMenu, true)
      clearTimeout(searchTimer)
    })

    return () => {
      const optionNodes = slots.default?.() || []
      options = readOptions(optionNodes)
      const selected = options.find(option => String(option.value) === String(props.modelValue))

      return h('div', {
        class: ['fas-choice', attrs.class],
        style: attrs.style,
        ref: root,
      }, [
        h('input', {
          type: 'hidden',
          name: props.name,
          value: props.modelValue,
          disabled: props.disabled,
        }),
        h('button', {
          ...Object.fromEntries(Object.entries(attrs).filter(([key]) => !['class', 'style'].includes(key))),
          class: ['fas-choice-trigger', { 'fas-choice-open': open.value }],
          type: 'button',
          disabled: props.disabled,
          ref: trigger,
          role: 'combobox',
          'aria-expanded': open.value,
          'aria-haspopup': 'listbox',
          'aria-required': props.required,
          'aria-controls': listID,
          'aria-activedescendant': open.value ? `${listID}-${activeIndex.value}` : undefined,
          onClick: () => open.value ? close() : show(),
          onKeydown: keydown,
        }, [
          h('span', { class: 'fas-choice-label' }, selected?.label || props.placeholder),
          h(FasIcon, { name: 'chevron-down', class: 'fas-choice-chevron' }),
        ]),
        open.value ? h(Teleport, { to: 'body' }, [
          h('div', {
            id: listID,
            class: 'fas-choice-menu',
            style: menuStyle.value,
            ref: menu,
            role: 'listbox',
            'aria-label': attrs['aria-label'] || props.placeholder,
          }, options.map((option, index) => h('button', {
            id: `${listID}-${index}`,
            key: `${String(option.value)}-${index}`,
            'data-index': index,
            class: ['fas-choice-option', {
              'fas-choice-selected': String(option.value) === String(props.modelValue),
              'fas-choice-focused': activeIndex.value === index,
            }],
            type: 'button',
            role: 'option',
            tabindex: -1,
            disabled: option.disabled,
            'aria-selected': String(option.value) === String(props.modelValue),
            onPointermove: () => {
              activeIndex.value = index
            },
            onMousedown: event => event.preventDefault(),
            onClick: () => choose(option),
          }, [
            h('span', option.label),
            String(option.value) === String(props.modelValue) ? h(FasIcon, { name: 'check' }) : null,
          ]))),
        ]) : null,
      ])
    }
  },
})
</script>

<style scoped>
.fas-choice {
  position: relative;
  width: 100%;
  min-width: 0;
}

.fas-choice-native {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  opacity: 0;
  pointer-events: none;
}

.fas-choice-trigger {
  display: flex;
  width: 100%;
  min-height: 2.85rem;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border: 1px solid var(--fas-border-strong);
  border-radius: 0.85rem;
  background: var(--fas-input);
  padding: 0.65rem 0.85rem;
  color: var(--fas-text);
  font: inherit;
  font-size: 0.8rem;
  text-align: left;
  cursor: pointer;
  transition: border-color 160ms ease, box-shadow 160ms ease;
}

.fas-choice-trigger:hover:not(:disabled),
.fas-choice-trigger:focus-visible,
.fas-choice-open {
  outline: none;
  border-color: var(--fas-accent-border);
  box-shadow: 0 0 0 3px var(--fas-accent-soft);
}

.fas-choice-trigger:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.fas-choice-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fas-choice-chevron {
  color: var(--fas-muted);
  transition: transform 160ms ease;
}

.fas-choice-open .fas-choice-chevron {
  transform: rotate(180deg);
}

.fas-choice-menu {
  position: fixed;
  z-index: 1500;
  overflow-y: auto;
  overscroll-behavior: contain;
  border: 1px solid var(--fas-accent-border);
  border-radius: 0.9rem;
  background: var(--fas-elevated);
  padding: 0.35rem;
  box-shadow: var(--fas-shadow-card);
  color: var(--fas-text);
  font-size: 0.8rem;
}

.fas-choice-option {
  display: flex;
  width: 100%;
  min-height: 2.65rem;
  align-items: center;
  justify-content: space-between;
  gap: 0.65rem;
  border: 0;
  border-radius: 0.65rem;
  background: transparent;
  padding: 0.65rem 0.75rem;
  color: var(--fas-text-soft);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.fas-choice-focused,
.fas-choice-option:hover,
.fas-choice-selected {
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.fas-choice-option:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
</style>
