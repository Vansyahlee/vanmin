<script setup>
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from 'vue'

import { useAuthStore } from '@/stores/auth'
import fasAiLogo from '@/assets/fas-ai/fas-ai-logo.png'
import fasAiEmblem from '@/assets/fas-ai/fas-ai-emblem.png'

const auth = useAuthStore()

const MAX_FILES = 10
const MAX_FILE_SIZE = 100 * 1024 * 1024
const MAX_CHATS = 50

const fileInput = ref(null)
const imageInput = ref(null)
const videoInput = ref(null)
const cameraInput = ref(null)
const messageList = ref(null)
const messageTextarea = ref(null)
const composerRoot = ref(null)

const historyVisible = ref(true)
const mobileHistoryVisible = ref(false)
const attachMenuOpen = ref(false)
const dragging = ref(false)
const inputText = ref('')
const selectedMode = ref('chat')
const attachments = ref([])
const conversations = ref([])
const activeConversationId = ref(null)
const notice = ref('')
const hasLoaded = ref(false)

let noticeTimeout = 0
let idSequence = 0
const objectUrls = new Set()

const makeId = () => {
  idSequence += 1
  return `${Date.now().toString(36)}-${idSequence.toString(36)}`
}

const role = computed(() => auth.user?.role || 'pegawai')
const currentUser = computed(() => auth.user?.name || 'Pengguna FAS')

const roleLabel = computed(() => ({
  admin: 'Executive Workspace',
  pegawai: 'Warehouse Workspace',
  finance: 'Finance Workspace',
}[role.value] || 'Internal Workspace'))

const storageKey = computed(() => {
  const identifier = auth.user?.id || auth.user?.email || 'guest'
  return `fas_ai_ui_preview_${String(identifier)}`
})

const activeConversation = computed(() =>
  conversations.value.find((item) => item.id === activeConversationId.value) || null,
)

const messages = computed(() => activeConversation.value?.messages || [])

const modes = {
  chat: {
    label: 'Chat',
    placeholder: 'Tanya apa saja tentang pekerjaanmu…',
  },
  image: {
    label: 'Buat gambar',
    placeholder: 'Deskripsikan gambar yang ingin dibuat…',
  },
  video: {
    label: 'Buat video',
    placeholder: 'Ceritakan adegan video yang ingin dibuat…',
  },
  document: {
    label: 'Buat dokumen',
    placeholder: 'Jelaskan dokumen Word, Excel, atau PDF yang dibutuhkan…',
  },
}

const promptSuggestions = computed(() => {
  if (role.value === 'finance') {
    return [
      { icon: 'document', text: 'Bantu analisis laporan keuangan' },
      { icon: 'document', text: 'Susun format laporan piutang' },
      { icon: 'document', text: 'Buat struktur rekap Excel' },
      { icon: 'fas', text: 'Jelaskan aging piutang' },
    ]
  }

  if (role.value === 'admin') {
    return [
      { icon: 'image', text: 'Rangkum aktivitas operasional' },
      { icon: 'document', text: 'Rancang dashboard manajemen' },
      { icon: 'fas', text: 'Jelaskan model Decision Tree' },
      { icon: 'document', text: 'Buat konsep laporan perusahaan' },
    ]
  }

  return [
    { icon: 'image', text: 'Prioritas kerja gudang hari ini' },
    { icon: 'arrow', text: 'Jelaskan proses picking dan packing' },
    { icon: 'document', text: 'Buat checklist pengecekan stok' },
    { icon: 'fas', text: 'Jelaskan rekomendasi distribusi' },
  ]
})

const formatTime = (value) => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat('id-ID', {
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

const formatFileSize = (bytes) => {
  if (!bytes) return '0 KB'
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

const getFileKind = (file) => {
  const name = String(file.name || '').toLowerCase()
  const mime = String(file.type || '').toLowerCase()

  if (mime.startsWith('image/')) return 'image'
  if (mime.startsWith('video/')) return 'video'
  if (mime.startsWith('audio/')) return 'audio'
  if (mime === 'application/pdf' || name.endsWith('.pdf')) return 'pdf'
  if (/\.(doc|docx|odt)$/.test(name)) return 'word'
  if (/\.(xls|xlsx|ods|csv)$/.test(name)) return 'excel'
  if (/\.(ppt|pptx)$/.test(name)) return 'slide'
  return 'file'
}

const fileIcon = (kind) => ({
  image: 'IMG',
  video: 'VID',
  audio: 'AUD',
  pdf: 'PDF',
  word: 'DOC',
  excel: 'XLS',
  slide: 'PPT',
  file: 'FILE',
}[kind] || 'FILE')

const glyph = {
  panel: 'M9 3H5a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V5a2 2 0 0 0-2-2H9Zm0 0v18',
  plus: 'M12 5v14M5 12h14',
  mic: 'M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Zm-7 9v1a7 7 0 0 0 14 0v-1M12 19v3m-4 0h8',
  stop: 'M6 6h12v12H6z',
  send: 'M12 19V5m-7 7 7-7 7 7',
  message: 'M21 11.5a8.4 8.4 0 0 1-9 8.4 9 9 0 0 1-4-.9L3 21l1.8-5a8.6 8.6 0 0 1-.8-4A8.5 8.5 0 0 1 12.5 3H13a8.5 8.5 0 0 1 8 8.5Z',
  trash: 'M3 6h18M8 6V4h8v2m3 0-1 14H6L5 6m5 4v7m4-7v7',
  chevron: 'm9 18 6-6-6-6',
  image: 'M4 4h16v16H4zM4 16l5-5 4 4 3-3 4 4M9 9h.01',
  video: 'M4 6h11v12H4zM15 10l5-3v10l-5-3',
  document: 'M7 3h7l4 4v14H7zM14 3v5h4M10 12h5m-5 4h5',
  camera: 'M4 8h3l2-3h6l2 3h3v12H4zM12 11a3 3 0 1 0 0 6 3 3 0 1 0 0-6Z',
  speaker: 'M11 5 6 9H3v6h3l5 4V5Zm4 4a5 5 0 0 1 0 6m2-9a9 9 0 0 1 0 12',
  arrow: 'M5 12h14m-6-6 6 6-6 6',
  close: 'M18 6 6 18M6 6l12 12',
  dots: 'M5 12h.01M12 12h.01M19 12h.01',
  paperclip: 'm21 11.5-8.5 8.5a6 6 0 0 1-8.5-8.5L13 3a4 4 0 0 1 5.7 5.7L9.5 18a2 2 0 0 1-2.8-2.8l8.5-8.5',
}

const modeOptions = [
  { key: 'chat', label: 'Chat', icon: 'message' },
  { key: 'image', label: 'Buat gambar', icon: 'image' },
  { key: 'video', label: 'Buat video', icon: 'video' },
  { key: 'document', label: 'Buat dokumen', icon: 'document' },
]

const listening = ref(false)
const voiceSupported = ref(false)
const speechSupported = ref(false)
const speakingId = ref(null)
let recognition = null
let voicePrefix = ''

const initVoice = () => {
  const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition
  voiceSupported.value = Boolean(SpeechRecognition)
  speechSupported.value = 'speechSynthesis' in window
  if (!SpeechRecognition) return

  recognition = new SpeechRecognition()
  recognition.lang = 'id-ID'
  recognition.interimResults = true
  recognition.continuous = false
  recognition.onstart = () => { listening.value = true }
  recognition.onresult = (event) => {
    const segments = []
    for (let i = 0; i < event.results.length; i += 1) {
      segments.push(event.results[i][0].transcript)
    }
    inputText.value = [voicePrefix, segments.join(' ').trim()].filter(Boolean).join(' ')
    nextTick(() => resizeTextarea())
  }
  recognition.onerror = (event) => {
    listening.value = false
    if (event.error !== 'aborted') showNotice('Mikrofon tidak tersedia atau izin suara ditolak oleh browser.')
  }
  recognition.onend = () => { listening.value = false }
}

const toggleMicrophone = () => {
  if (!recognition) {
    showNotice('Browser ini belum mendukung Speech Recognition.')
    return
  }
  if (listening.value) {
    recognition.stop()
    return
  }
  voicePrefix = inputText.value.trim()
  try { recognition.start() } catch { showNotice('Mikrofon belum siap. Coba lagi.') }
}

const speakMessage = (message) => {
  if (!speechSupported.value || !message?.text) return
  const synth = window.speechSynthesis
  if (speakingId.value === message.id) {
    synth.cancel()
    speakingId.value = null
    return
  }
  synth.cancel()
  const utterance = new SpeechSynthesisUtterance(message.text)
  utterance.lang = 'id-ID'
  utterance.rate = 1
  utterance.onend = () => { speakingId.value = null }
  utterance.onerror = () => { speakingId.value = null }
  speakingId.value = message.id
  synth.speak(utterance)
}

const showNotice = (message) => {
  notice.value = message
  window.clearTimeout(noticeTimeout)
  noticeTimeout = window.setTimeout(() => { notice.value = '' }, 5000)
}

const serialize = () => conversations.value.slice(0, MAX_CHATS).map((chat) => ({
  id: chat.id,
  title: chat.title,
  updatedAt: chat.updatedAt,
  messages: chat.messages.map((message) => ({
    id: message.id,
    from: message.from,
    text: message.text,
    mode: message.mode,
    createdAt: message.createdAt,
    files: (message.files || []).map((file) => ({
      id: file.id,
      name: file.name,
      size: file.size,
      kind: file.kind,
    })),
  })),
}))

const saveChats = () => {
  if (!hasLoaded.value) return
  try {
    sessionStorage.setItem(storageKey.value, JSON.stringify(serialize()))
  } catch {
    showNotice('Penyimpanan riwayat lokal browser penuh. Chat baru tetap dapat digunakan.')
  }
}

const loadChats = () => {
  try {
    const stored = JSON.parse(sessionStorage.getItem(storageKey.value) || '[]')
    if (Array.isArray(stored)) {
      conversations.value = stored.slice(0, MAX_CHATS).filter(
        (item) => item && typeof item.id === 'string' && Array.isArray(item.messages),
      )
    }
  } catch {
    conversations.value = []
  }
  activeConversationId.value = conversations.value[0]?.id || null
  hasLoaded.value = true
}

const releaseFile = (item) => {
  if (item?.url && objectUrls.has(item.url)) {
    URL.revokeObjectURL(item.url)
    objectUrls.delete(item.url)
  }
}

const clearPendingFiles = () => {
  attachments.value.forEach(releaseFile)
  attachments.value = []
}

const toggleHistory = () => {
  if (window.matchMedia('(max-width: 900px)').matches) {
    mobileHistoryVisible.value = !mobileHistoryVisible.value
    historyVisible.value = true
  } else {
    historyVisible.value = !historyVisible.value
  }
}

const hideHistory = () => {
  mobileHistoryVisible.value = false
  if (!window.matchMedia('(max-width: 900px)').matches) {
    historyVisible.value = false
  }
}

const startNewChat = () => {
  clearPendingFiles()
  activeConversationId.value = null
  inputText.value = ''
  selectedMode.value = 'chat'
  attachMenuOpen.value = false
  mobileHistoryVisible.value = false
  nextTick(() => messageTextarea.value?.focus())
}

const switchConversation = (id) => {
  if (id === activeConversationId.value) return
  clearPendingFiles()
  activeConversationId.value = id
  mobileHistoryVisible.value = false
  attachMenuOpen.value = false
  scrollToBottom()
}

const deleteConversation = (id) => {
  const target = conversations.value.find((chat) => chat.id === id)
  target?.messages.forEach((message) => (message.files || []).forEach(releaseFile))
  conversations.value = conversations.value.filter((chat) => chat.id !== id)
  if (activeConversationId.value === id) {
    activeConversationId.value = conversations.value[0]?.id || null
  }
  saveChats()
}

const scrollToBottom = async () => {
  await nextTick()
  if (messageList.value) {
    messageList.value.scrollTop = messageList.value.scrollHeight
  }
}

const triggerPicker = (target) => {
  attachMenuOpen.value = false
  target?.click()
}

const chooseMode = (mode) => {
  selectedMode.value = mode
  attachMenuOpen.value = false
  messageTextarea.value?.focus()

  if (mode !== 'chat') {
    showNotice(`${modes[mode].label}: belum tersedia. Fitur ini akan aktif setelah layanan AI tersambung.`)
  }
}

const addFiles = (fileList) => {
  if (!fileList?.length) return
  const available = MAX_FILES - attachments.value.length
  let rejected = 0

  for (const file of Array.from(fileList).slice(0, Math.max(available, 0))) {
    if (file.size > MAX_FILE_SIZE) {
      rejected += 1
      continue
    }

    const kind = getFileKind(file)
    let url = ''

    if (kind === 'image' || kind === 'video' || kind === 'audio') {
      url = URL.createObjectURL(file)
      objectUrls.add(url)
    }

    attachments.value.push({
      id: makeId(),
      file,
      name: file.name,
      size: file.size,
      kind,
      url,
    })
  }

  if (available <= 0 || fileList.length > available) {
    showNotice(`Maksimal ${MAX_FILES} lampiran dalam satu pesan.`)
  } else if (rejected) {
    showNotice(`${rejected} file terlalu besar. Batas per file ${formatFileSize(MAX_FILE_SIZE)}.`)
  }
}

const onFileSelect = (event) => {
  addFiles(event.target.files)
  event.target.value = ''
}

const removePending = (id) => {
  const item = attachments.value.find((file) => file.id === id)
  releaseFile(item)
  attachments.value = attachments.value.filter((file) => file.id !== id)
}

const onDrop = (event) => {
  dragging.value = false
  addFiles(event.dataTransfer?.files)
}

const onPaste = (event) => {
  const clipboardFiles = Array.from(event.clipboardData?.files || [])
  if (clipboardFiles.length) {
    event.preventDefault()
    addFiles(clipboardFiles)
  }
}

const resizeTextarea = (event) => {
  const target = event?.target || messageTextarea.value
  if (!target) return
  target.style.height = 'auto'
  target.style.height = `${Math.min(target.scrollHeight, 180)}px`
}

const onComposerKeydown = (event) => {
  if (event.key === 'Enter' && !event.shiftKey && !event.isComposing) {
    event.preventDefault()
    sendMessage()
  }
}

const createConversation = () => {
  const chat = {
    id: makeId(),
    title: 'Percakapan baru',
    updatedAt: new Date().toISOString(),
    messages: [],
  }
  conversations.value.unshift(chat)
  activeConversationId.value = chat.id
  return chat
}

const canSend = computed(() =>
  Boolean(inputText.value.trim() || attachments.value.length),
)

const sendMessage = () => {
  if (!canSend.value) return

  const text = inputText.value.trim()
  const chat = activeConversation.value || createConversation()
  const mode = selectedMode.value
  const selected = [...attachments.value]

  chat.messages.push({
    id: makeId(),
    from: 'user',
    text,
    mode,
    files: selected,
    createdAt: new Date().toISOString(),
  })

  if (chat.messages.filter((message) => message.from === 'user').length === 1) {
    chat.title = (text || selected[0]?.name || 'Percakapan baru').slice(0, 44)
  }

  chat.messages.push({
    id: makeId(),
    from: 'system',
    text: mode === 'chat'
      ? 'Layanan percakapan AI belum aktif. Pesan dan lampiran hanya tersimpan sementara di perangkat ini; belum ada analisis otomatis.'
      : `Permintaan “${modes[mode].label}” belum dapat diproses karena layanan pembuatan konten belum tersedia.`,
    mode,
    files: [],
    createdAt: new Date().toISOString(),
  })

  chat.updatedAt = new Date().toISOString()
  conversations.value = [
    chat,
    ...conversations.value.filter((item) => item.id !== chat.id),
  ].slice(0, MAX_CHATS)

  attachments.value = []
  inputText.value = ''
  saveChats()
  nextTick(() => {
    resizeTextarea()
    scrollToBottom()
    messageTextarea.value?.focus()
  })
}

const fillSuggestion = (text) => {
  inputText.value = text
  nextTick(() => {
    resizeTextarea()
    messageTextarea.value?.focus()
  })
}

const outsideClick = (event) => {
  if (attachMenuOpen.value && !composerRoot.value?.contains(event.target)) {
    attachMenuOpen.value = false
  }
}

const onEscape = (event) => {
  if (event.key === 'Escape') {
    attachMenuOpen.value = false
    mobileHistoryVisible.value = false
  }
}

watch(
  () => auth.user?.id || auth.user?.email,
  () => {
    if (!hasLoaded.value) return
    conversations.value.forEach((chat) => {
      chat.messages.forEach((message) => (message.files || []).forEach(releaseFile))
    })
    clearPendingFiles()
    hasLoaded.value = false
    loadChats()
  },
)

onMounted(() => {
  loadChats()
  initVoice()
  window.addEventListener('pointerdown', outsideClick)
  window.addEventListener('keydown', onEscape)
})

onBeforeUnmount(() => {
  try { recognition?.abort() } catch {}
  if (speechSupported.value) window.speechSynthesis.cancel()
  window.clearTimeout(noticeTimeout)
  window.removeEventListener('pointerdown', outsideClick)
  window.removeEventListener('keydown', onEscape)
  clearPendingFiles()
  conversations.value.forEach((chat) => {
    chat.messages.forEach((message) => (message.files || []).forEach(releaseFile))
  })
})
</script>
<template>
  <div
    class="fas-ai-workspace"
    :class="{ 'history-collapsed': !historyVisible, 'is-dragging': dragging }"
    @dragenter.prevent="dragging = true"
    @dragover.prevent
  >
    <div v-if="mobileHistoryVisible" class="mobile-scrim" @click="mobileHistoryVisible = false"></div>

    <aside
      class="ai-history"
      :class="{ 'is-mobile-open': mobileHistoryVisible }"
      aria-label="Riwayat percakapan"
      :aria-hidden="!historyVisible && !mobileHistoryVisible"
      :inert="!historyVisible && !mobileHistoryVisible"
    >
      <div class="history-head">
        <div class="history-brand">
          <span class="brand-emblem" aria-hidden="true">
            <img :src="fasAiEmblem" alt="" draggable="false" />
          </span>
          <span class="history-brand-name"><strong>FAS AI</strong><small>Workspace internal</small></span>
        </div>
        <button class="icon-button" type="button" title="Tutup riwayat" aria-label="Tutup riwayat" @click="hideHistory">
          <svg viewBox="0 0 24 24"><path :d="glyph.panel"/></svg>
        </button>
      </div>

      <button class="new-chat" type="button" @click="startNewChat">
        <svg viewBox="0 0 24 24"><path :d="glyph.plus"/></svg>
        <span>Percakapan baru</span>
      </button>

      <div class="history-label">Terbaru</div>
      <div class="history-list">
        <p v-if="!conversations.length" class="history-empty">Belum ada percakapan. Mulai dengan mengetik pesan.</p>
        <div v-for="chat in conversations" :key="chat.id" class="history-entry" :class="{ active: activeConversationId === chat.id }">
          <button class="history-select" type="button" :title="chat.title" @click="switchConversation(chat.id)">
            <svg viewBox="0 0 24 24"><path :d="glyph.message"/></svg>
            <span>{{ chat.title }}</span>
          </button>
          <button class="history-remove" type="button" :aria-label="`Hapus ${chat.title}`" title="Hapus percakapan" @click="deleteConversation(chat.id)">
            <svg viewBox="0 0 24 24"><path :d="glyph.trash"/></svg>
          </button>
        </div>
      </div>
      <div class="history-foot">
        <span class="history-foot-icon"><img :src="fasAiEmblem" alt="" draggable="false" /></span>
        <div><strong>FAS Artificial Intelligence</strong><small>Ruang kerja internal PT. FAS</small></div>
      </div>
    </aside>

    <section class="ai-main" aria-label="FAS AI Chat" @drop.prevent="onDrop" @dragleave.prevent="dragging = false">
      <div class="ai-ambient" aria-hidden="true"></div>
      <header class="chat-header">
        <div class="chat-head-left">
          <button class="icon-button toggle-history" type="button" :title="historyVisible ? 'Sembunyikan riwayat' : 'Tampilkan riwayat'" :aria-label="historyVisible ? 'Sembunyikan riwayat' : 'Tampilkan riwayat'" @click="toggleHistory">
            <svg viewBox="0 0 24 24"><path :d="glyph.panel"/></svg>
          </button>
          <span class="header-brand">FAS AI <span class="header-chevron">⌄</span></span>
        </div>
        <div class="chat-head-right">
          <span class="head-role">{{ roleLabel }}</span>
          <button class="icon-button" type="button" title="Percakapan baru" aria-label="Percakapan baru" @click="startNewChat">
            <svg viewBox="0 0 24 24"><path :d="glyph.plus"/></svg>
          </button>
        </div>
      </header>

      <div v-if="notice" class="ai-notice" role="status">{{ notice }}</div>
      <div v-if="dragging" class="drop-overlay" @drop.prevent="onDrop" @dragleave.prevent="dragging = false">
        <svg viewBox="0 0 24 24"><path :d="glyph.paperclip"/></svg>
        <strong>Lepaskan file untuk dilampirkan</strong>
        <span>Foto, video, PDF, Word, Excel, dan berkas lain</span>
      </div>

      <div ref="messageList" class="chat-scroll" :class="{ 'is-empty': !messages.length }">
        <div v-if="!messages.length" class="welcome-content">
          <div class="welcome-mark" aria-label="Logo premium FAS AI" role="img">
            <img :src="fasAiLogo" alt="FAS" draggable="false" />
          </div>
          <p class="welcome-brand">FAS ARTIFICIAL INTELLIGENCE</p>
          <h1>Apa yang bisa <span>FAS AI</span> bantu?</h1>
          <p class="welcome-subtitle">Tulis pertanyaan, lampirkan file, atau gunakan suara. Semua berada dalam satu ruang kerja PT. FAS.</p>
          <div class="welcome-prompts">
            <button v-for="suggestion in promptSuggestions" :key="suggestion.text" type="button" @click="fillSuggestion(suggestion.text)">
              <span class="prompt-icon">
                <img v-if="suggestion.icon === 'fas'" :src="fasAiEmblem" alt="" draggable="false" class="prompt-fas-emblem" />
                <svg v-else viewBox="0 0 24 24"><path :d="glyph[suggestion.icon] || glyph.document"/></svg>
              </span>
              <span>{{ suggestion.text }}</span>
              <svg class="prompt-arrow" viewBox="0 0 24 24"><path :d="glyph.arrow"/></svg>
            </button>
          </div>
        </div>

        <div v-else class="messages-inner">
          <article v-for="message in messages" :key="message.id" class="chat-message" :class="`message-${message.from}`">
            <span v-if="message.from !== 'user'" class="assistant-avatar" aria-hidden="true"><img :src="fasAiEmblem" alt="" draggable="false" /></span>
            <div class="message-body">
              <div v-if="message.text" class="message-text">{{ message.text }}</div>
              <div v-if="message.files?.length" class="message-files">
                <div v-for="file in message.files" :key="file.id" class="sent-file">
                  <img v-if="file.kind === 'image' && file.url" :src="file.url" :alt="file.name" />
                  <video v-else-if="file.kind === 'video' && file.url" :src="file.url" controls preload="metadata"></video>
                  <span v-else class="file-badge">{{ fileIcon(file.kind) }}</span>
                  <span class="file-description"><strong>{{ file.name }}</strong><small>{{ formatFileSize(file.size) }}</small></span>
                </div>
              </div>
              <div class="message-actions">
                <span>{{ message.from === 'user' ? 'Anda' : 'FAS AI' }} · {{ formatTime(message.createdAt) }}</span>
                <button v-if="message.from !== 'user' && speechSupported && message.text" class="speak-button" type="button" :title="speakingId === message.id ? 'Hentikan suara' : 'Dengarkan balasan'" :aria-label="speakingId === message.id ? 'Hentikan suara' : 'Dengarkan balasan'" @click="speakMessage(message)">
                  <svg viewBox="0 0 24 24"><path :d="speakingId === message.id ? glyph.stop : glyph.speaker"/></svg>
                </button>
              </div>
            </div>
          </article>
        </div>
      </div>

      <div ref="composerRoot" class="composer-area">
        <div class="composer-wrap">
          <div v-if="selectedMode !== 'chat'" class="selected-mode">
            <svg viewBox="0 0 24 24"><path :d="glyph[modeOptions.find((item) => item.key === selectedMode)?.icon || 'document']"/></svg>
            <strong>{{ modes[selectedMode].label }}</strong>
            <small>Fitur belum aktif</small>
            <button type="button" title="Hapus mode" aria-label="Hapus mode" @click="selectedMode = 'chat'"><svg viewBox="0 0 24 24"><path :d="glyph.close"/></svg></button>
          </div>

          <div v-if="attachments.length" class="pending-files">
            <div v-for="file in attachments" :key="file.id" class="pending-file">
              <img v-if="file.kind === 'image' && file.url" :src="file.url" :alt="file.name" />
              <video v-else-if="file.kind === 'video' && file.url" :src="file.url" muted preload="metadata"></video>
              <span v-else class="file-badge">{{ fileIcon(file.kind) }}</span>
              <span class="pending-name" :title="file.name">{{ file.name }}<small>{{ formatFileSize(file.size) }}</small></span>
              <button type="button" :aria-label="`Hapus ${file.name}`" @click="removePending(file.id)"><svg viewBox="0 0 24 24"><path :d="glyph.close"/></svg></button>
            </div>
          </div>

          <form class="composer-form" @submit.prevent="sendMessage">
            <textarea
              ref="messageTextarea"
              v-model="inputText"
              :placeholder="modes[selectedMode].placeholder"
              rows="1"
              aria-label="Pesan untuk FAS AI"
              @input="resizeTextarea"
              @keydown="onComposerKeydown"
              @paste="onPaste"
            ></textarea>
            <div class="composer-controls">
              <div class="composer-left">
                <div class="attach-anchor">
                  <button class="tool-button" type="button" title="Lampirkan atau pilih alat" aria-label="Lampirkan atau pilih alat" :aria-expanded="attachMenuOpen" @click="attachMenuOpen = !attachMenuOpen">
                    <svg viewBox="0 0 24 24"><path :d="glyph.plus"/></svg>
                  </button>
                  <div v-if="attachMenuOpen" class="attach-menu" role="menu">
                    <div class="menu-label">UNGGAH DARI PERANGKAT</div>
                    <button type="button" role="menuitem" @click="triggerPicker(imageInput)"><svg viewBox="0 0 24 24"><path :d="glyph.image"/></svg> Gambar & foto</button>
                    <button type="button" role="menuitem" @click="triggerPicker(videoInput)"><svg viewBox="0 0 24 24"><path :d="glyph.video"/></svg> Video</button>
                    <button type="button" role="menuitem" @click="triggerPicker(fileInput)"><svg viewBox="0 0 24 24"><path :d="glyph.paperclip"/></svg> PDF, Word, Excel & lainnya</button>
                    <button type="button" role="menuitem" @click="triggerPicker(cameraInput)"><svg viewBox="0 0 24 24"><path :d="glyph.camera"/></svg> Kamera perangkat</button>
                    <div class="menu-divider"></div>
                    <div class="menu-label">ALAT KREATIF</div>
                    <button v-for="mode in modeOptions.slice(1)" :key="mode.key" type="button" role="menuitem" @click="chooseMode(mode.key)">
                      <svg viewBox="0 0 24 24"><path :d="glyph[mode.icon]"/></svg>{{ mode.label }}<small>Belum aktif</small>
                    </button>
                  </div>
                </div>
                <button class="mode-trigger" type="button" title="Pilih mode" @click="attachMenuOpen = !attachMenuOpen">
                  <img :src="fasAiEmblem" class="mode-fas-emblem" alt="" draggable="false" />
                  <span>{{ modes[selectedMode].label }}</span>
                  <span class="mode-caret">⌄</span>
                </button>
              </div>
              <div class="composer-right">
                <button class="tool-button mic-button" type="button" :disabled="!voiceSupported" :class="{ 'is-listening': listening }" :title="!voiceSupported ? 'Mikrofon tidak didukung browser' : listening ? 'Hentikan mikrofon' : 'Input suara'" :aria-label="listening ? 'Hentikan mikrofon' : 'Input suara'" @click="toggleMicrophone">
                  <svg viewBox="0 0 24 24"><path :d="listening ? glyph.stop : glyph.mic"/></svg>
                </button>
                <button class="send-button" type="submit" :disabled="!canSend" aria-label="Kirim pesan" title="Kirim pesan">
                  <svg viewBox="0 0 24 24"><path :d="glyph.send"/></svg>
                </button>
              </div>
            </div>
          </form>
        </div>
        <p class="composer-footnote">
          <template v-if="listening">Mendengarkan suara…</template>
          <template v-else>Layanan AI belum aktif · Pesan dan lampiran hanya disimpan sementara di perangkat ini.</template>
        </p>
      </div>
    </section>

    <input ref="fileInput" type="file" class="file-input" multiple accept=".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.csv,.odt,.ods,.zip,.json,image/*,video/*,audio/*" @change="onFileSelect" />
    <input ref="imageInput" type="file" class="file-input" multiple accept="image/*" @change="onFileSelect" />
    <input ref="videoInput" type="file" class="file-input" multiple accept="video/*" @change="onFileSelect" />
    <input ref="cameraInput" type="file" class="file-input" accept="image/*" capture="environment" @change="onFileSelect" />
  </div>
</template>
<style scoped>
.fas-ai-workspace {
  --ai-line: var(--fas-border);
  --ai-text: var(--fas-text);
  --ai-muted: var(--fas-muted);
  --ai-accent: var(--fas-accent);
  --ai-soft: var(--fas-accent-soft);
  --ai-glass: color-mix(in srgb, var(--fas-card) 82%, transparent);
  display: grid;
  grid-template-columns: 258px minmax(0, 1fr);
  width: 100%;
  height: calc(100dvh - 96px);
  min-height: 500px;
  overflow: hidden;
  position: relative;
  color: var(--ai-text);
  background: var(--fas-bg);
  transition: grid-template-columns .34s cubic-bezier(.22,1,.36,1);
}
.fas-ai-workspace.history-collapsed { grid-template-columns: 0px minmax(0, 1fr); }
.fas-ai-workspace *, .fas-ai-workspace *::before, .fas-ai-workspace *::after { box-sizing: border-box; }
.fas-ai-workspace button, .fas-ai-workspace textarea { font: inherit; }
.fas-ai-workspace button { cursor: pointer; }
.fas-ai-workspace svg { display: block; width: 20px; height: 20px; fill: none; stroke: currentColor; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
.fas-ai-workspace button:focus-visible, .fas-ai-workspace textarea:focus-visible { outline: 2px solid var(--fas-accent); outline-offset: 3px; }
.ai-history {
  min-width: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--ai-line);
  background: color-mix(in srgb, var(--fas-surface) 80%, var(--fas-bg));
  transition: opacity .18s ease;
}
.history-collapsed .ai-history { opacity: 0; border-right: 0; pointer-events: none; }
.history-head { min-width: 246px; padding: 19px 14px 10px 19px; display: flex; gap: 6px; align-items: center; justify-content: space-between; }
.history-brand { display: flex; align-items: center; min-width: 0; gap: 10px; }
.brand-emblem { display: grid; place-items: center; width: 38px; height: 38px; flex: 0 0 38px; border-radius: 13px; background: color-mix(in srgb, var(--fas-accent) 11%, var(--fas-card)); box-shadow: inset 0 0 0 1px var(--fas-accent-border); }
.brand-emblem img { width: 35px; height: 35px; object-fit: contain; display: block; filter: drop-shadow(0 3px 6px #1472bf22); }
.history-brand-name { display: flex; flex-direction: column; gap: 3px; }
.history-brand-name strong { font-size: 14px; font-weight: 800; letter-spacing: -.025em; }
.history-brand-name small { font-size: 10px; color: var(--ai-muted); }
.icon-button { width: 36px; height: 36px; display: grid; place-items: center; border: 0; border-radius: 11px; color: var(--fas-text-soft); background: transparent; flex: 0 0 auto; transition: background .17s; }
.icon-button:hover { background: var(--fas-hover); }
.icon-button svg { width: 19px; height: 19px; }
.new-chat { display: flex; align-items: center; justify-content: flex-start; gap: 12px; margin: 17px 15px 25px; padding: 12px 13px; border: 1px solid var(--fas-accent-border); color: var(--fas-accent-text); font-size: 12px !important; font-weight: 750 !important; border-radius: 13px; background: var(--ai-soft); min-width: 225px; transition: background .2s, transform .2s; }
.new-chat:hover { transform: translateY(-1px); background: color-mix(in srgb, var(--fas-accent) 16%, var(--fas-card)); }
.new-chat svg { width: 17px; height: 17px; }
.history-label { padding: 0 21px 12px; min-width: 246px; color: var(--ai-muted); font-size: 11px; font-weight: 700; }
.history-list { min-width: 246px; padding: 0 12px 18px; flex: 1; overflow-y: auto; }
.history-entry { display: flex; align-items: center; min-width: 0; border-radius: 12px; margin-bottom: 4px; transition: background .2s; }
.history-entry:hover, .history-entry.active { background: var(--fas-hover); }
.history-entry.active { background: var(--fas-accent-soft); }
.history-select { display: flex; align-items: center; gap: 10px; flex: 1; min-width: 0; color: var(--fas-text-soft); background: transparent; padding: 12px 8px; border: 0; text-align: left; font-size: 12px !important; }
.history-select svg { flex-shrink: 0; width: 16px; height: 16px; color: var(--ai-muted); }
.history-select span { overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.history-remove { width: 28px; height: 28px; border: 0; border-radius: 8px; display: grid; place-items: center; color: var(--ai-muted); background: transparent; flex: 0 0 auto; }
.history-remove:hover { color: var(--fas-danger); background: var(--fas-hover); }
.history-remove svg { width: 15px; height: 15px; }
.history-empty { padding: 15px 10px; color: var(--ai-muted); font-size: 12px; line-height: 1.7; }
.history-foot { display: flex; min-width: 246px; align-items: center; gap: 10px; border-top: 1px solid var(--ai-line); padding: 18px 17px; }
.history-foot-icon { background: var(--fas-accent-soft); color: var(--fas-accent-text); width: 31px; height: 31px; display: grid; place-items: center; border-radius: 12px; }
.history-foot-icon img { width: 29px; height: 29px; object-fit: contain; display: block; }
.history-foot div { display: flex; flex-direction: column; gap: 4px; }
.history-foot strong { font-size: 11px; font-weight: 750; }
.history-foot small { font-size: 10px; color: var(--ai-muted); }
.ai-main { display: flex; position: relative; flex-direction: column; overflow: hidden; min-width: 0; background: var(--fas-bg); isolation: isolate; }
.ai-main::before { content: ''; position: absolute; z-index: -2; inset: 0; pointer-events: none; background: radial-gradient(ellipse 70% 50% at 52% 52%, color-mix(in srgb, var(--fas-accent) 13%, transparent), transparent 78%), radial-gradient(ellipse 32% 34% at 84% 17%, color-mix(in srgb, #7657dc 11%, transparent), transparent 100%), var(--fas-bg); }
.ai-ambient { position: absolute; z-index: -1; pointer-events: none; width: 55%; height: 55%; left: 23%; top: 26%; border-radius: 50%; background: color-mix(in srgb, var(--fas-accent) 7%, transparent); filter: blur(100px); }
:global(html[data-theme="light"]) .ai-main::before { background: radial-gradient(ellipse 72% 50% at 50% 42%, #dce5ffb7, transparent 83%), radial-gradient(ellipse 45% 50% at 100% 20%, #d9f8ee70, transparent 100%), #f7f9ff; }
:global(html[data-theme="dark"]) .ai-main::before { background: radial-gradient(ellipse 72% 60% at 50% 52%, #14265a90, transparent 78%), radial-gradient(ellipse 45% 60% at 100% 20%, #32205440, transparent), #080b14; }
@media (prefers-color-scheme: light) { :global(html[data-theme="system"]) .ai-main::before { background: radial-gradient(ellipse 72% 50% at 50% 42%, #dce5ffb7, transparent 83%), #f7f9ff; } }
@media (prefers-color-scheme: dark) { :global(html[data-theme="system"]) .ai-main::before { background: radial-gradient(ellipse 72% 60% at 50% 52%, #14265a90, transparent 78%), #080b14; } }
.chat-header { position: relative; z-index: 5; height: 63px; min-height: 63px; display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 0 clamp(15px, 2.3vw, 35px); }
.chat-head-left, .chat-head-right { display: flex; align-items: center; gap: 12px; min-width: 0; }
.header-brand { font-size: 15px; font-weight: 780; letter-spacing: -.025em; white-space: nowrap; }
.header-chevron { color: var(--ai-muted); font-size: 14px; margin-left: 4px; }
.head-role { font-size: 11px; color: var(--ai-muted); white-space: nowrap; }
.chat-scroll { flex: 1; min-height: 0; width: 100%; overflow-y: auto; overflow-x: hidden; overscroll-behavior: contain; padding: 12px clamp(15px, 3vw, 56px) 35px; }
.chat-scroll.is-empty { display: flex; align-items: center; justify-content: center; }
.welcome-content { width: min(100%, 990px); margin: auto; text-align: center; padding: 5px 0 22px; animation: intro-appear .5s ease both; }
.history-collapsed .welcome-content { width: min(100%, 1110px); }
.welcome-mark { display: grid; place-items: center; width: min(270px, 58vw); height: 119px; margin: 0 auto 14px; padding: 5px 10px; border-radius: 24px; background: linear-gradient(125deg, rgba(246,251,255,.96), rgba(226,242,255,.90)); box-shadow: 0 13px 42px #2563eb14, 0 0 76px color-mix(in srgb, var(--fas-accent) 14%, transparent); }
.welcome-mark img { display: block; width: 100%; height: 100%; object-fit: contain; }
.welcome-brand { color: var(--fas-accent-text); font-size: 10px; font-weight: 800; letter-spacing: .19em; margin: 0 0 15px; }
.welcome-content h1 { margin: 0; font-size: clamp(27px, 3.7vw, 54px); line-height: 1.14; font-weight: 690; letter-spacing: -.055em; color: var(--fas-text); }
.welcome-content h1 span { background: linear-gradient(100deg, #12B3DA 0%, #527CFF 34%, #9873F0 66%, #19B99B 100%); -webkit-background-clip: text; background-clip: text; color: transparent; font-weight: 790; }
.welcome-subtitle { color: var(--ai-muted); font-size: clamp(12px, 1vw, 14px); line-height: 1.8; margin: 16px auto 0; max-width: 670px; }
.welcome-prompts { margin: 35px auto 0; display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 11px; width: min(100%, 780px); }
.welcome-prompts button { display: flex; align-items: center; gap: 11px; min-width: 0; min-height: 57px; padding: 11px 15px; border-radius: 16px; border: 1px solid var(--fas-border); background: color-mix(in srgb, var(--fas-card) 68%, transparent); text-align: left; color: var(--fas-text-soft); font-size: 11.5px !important; transition: transform .2s, border-color .2s, background .2s; }
.welcome-prompts button:hover { transform: translateY(-2px); border-color: var(--fas-accent-border); background: var(--fas-card); }
.prompt-icon { color: var(--fas-accent-text); background: var(--fas-accent-soft); width: 32px; height: 32px; display: grid; place-items: center; border-radius: 11px; flex-shrink: 0; }
.prompt-icon svg { width: 17px; height: 17px; }
.prompt-fas-emblem { display: block; width: 27px; height: 27px; object-fit: contain; filter: drop-shadow(0 2px 3px #124cff16); }
.prompt-arrow { width: 15px !important; height: 15px !important; margin-left: auto; flex-shrink: 0; color: var(--ai-muted); }
.messages-inner { width: min(100%, 990px); margin: 20px auto; display: flex; flex-direction: column; gap: 29px; }
.history-collapsed .messages-inner { width: min(100%, 1110px); }
.chat-message { display: flex; gap: 13px; align-items: flex-start; }
.chat-message.message-user { justify-content: flex-end; }
.assistant-avatar { width: 31px; height: 31px; flex: 0 0 31px; border-radius: 11px; display: grid; place-items: center; background: var(--fas-accent-soft); color: var(--fas-accent-text); }
.assistant-avatar img { width: 29px; height: 29px; object-fit: contain; display: block; }
.message-body { min-width: 0; max-width: min(100%, 890px); }
.message-user .message-body { max-width: min(85%, 800px); }
.message-text { font-size: 13px; line-height: 1.85; white-space: pre-wrap; overflow-wrap: anywhere; color: var(--fas-text-soft); }
.message-user .message-text { padding: 12px 17px; border-radius: 18px 18px 5px 18px; background: color-mix(in srgb, var(--fas-accent) 13%, var(--fas-card)); color: var(--fas-text); border: 1px solid var(--fas-accent-border); }
.message-actions { display: flex; align-items: center; gap: 12px; min-height: 30px; color: var(--ai-muted); font-size: 10px; padding-top: 6px; }
.message-user .message-actions { justify-content: flex-end; }
.speak-button { display: grid; place-items: center; width: 29px; height: 29px; border: 0; background: transparent; color: var(--ai-muted); border-radius: 9px; }
.speak-button:hover { background: var(--fas-hover); color: var(--fas-text); }
.speak-button svg { width: 17px; height: 17px; }
.message-files { display: flex; flex-wrap: wrap; gap: 9px; margin-top: 9px; }
.sent-file { display: flex; align-items: center; gap: 10px; max-width: 300px; min-width: 180px; padding: 9px; border: 1px solid var(--ai-line); background: var(--fas-card); border-radius: 13px; }
.sent-file img, .sent-file video { width: 65px; height: 55px; border-radius: 9px; object-fit: cover; }
.file-badge { display: grid; place-items: center; flex: 0 0 40px; height: 42px; border-radius: 10px; background: var(--ai-soft); color: var(--fas-accent-text); font-size: 10px; font-weight: 900; }
.file-description { min-width: 0; }
.file-description strong { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 11px; }
.file-description small { display: block; color: var(--ai-muted); font-size: 10px; padding-top: 6px; }
.composer-area { position: relative; z-index: 6; width: 100%; flex: 0 0 auto; padding: 0 clamp(15px, 3vw, 56px) 16px; }
.composer-wrap { position: relative; width: min(100%, 990px); margin: 0 auto; border-radius: 28px; border: 1px solid var(--fas-border-strong); background: var(--ai-glass); box-shadow: 0 16px 52px color-mix(in srgb, var(--fas-bg) 35%, transparent); backdrop-filter: blur(28px); -webkit-backdrop-filter: blur(28px); transition: width .34s cubic-bezier(.22,1,.36,1), border-color .2s, box-shadow .2s; }
.history-collapsed .composer-wrap { width: min(100%, 1110px); }
.composer-wrap:focus-within { border-color: var(--fas-accent-border); box-shadow: 0 16px 52px color-mix(in srgb, var(--fas-accent) 12%, transparent); }
.composer-form textarea { display: block; width: 100%; min-height: 53px; max-height: 180px; resize: none; border: none; background: transparent; color: var(--fas-text); padding: 19px 21px 6px; font-size: 13px; line-height: 1.7; outline: none !important; overflow-y: auto; }
.composer-form textarea::placeholder { color: var(--ai-muted); }
.composer-controls { display: flex; justify-content: space-between; align-items: center; gap: 10px; padding: 5px 12px 12px; }
.composer-left, .composer-right { display: flex; align-items: center; gap: 7px; }
.attach-anchor { position: relative; }
.tool-button, .send-button { width: 38px; height: 38px; display: grid; place-items: center; flex: 0 0 auto; border-radius: 50%; border: none; background: transparent; color: var(--fas-text-soft); transition: background .2s, transform .2s; }
.tool-button:hover:not(:disabled) { background: var(--fas-hover); transform: translateY(-1px); }
.tool-button:disabled { opacity: .4; cursor: not-allowed; }
.tool-button svg, .send-button svg { width: 21px; height: 21px; }
.mic-button.is-listening { background: var(--fas-danger-soft); color: var(--fas-danger); animation: pulse-mic 1.2s ease-in-out infinite; }
.send-button { background: var(--fas-accent); color: #fff; }
.send-button:hover:not(:disabled) { background: var(--fas-accent-hover); }
.send-button:disabled { opacity: .38; cursor: not-allowed; }
.mode-trigger { display: flex; align-items: center; gap: 7px; padding: 8px 11px; border: none; background: transparent; color: var(--fas-text-soft); border-radius: 999px; font-size: 12px !important; }
.mode-trigger svg { color: var(--fas-accent-text); width: 15px; height: 15px; }
.mode-trigger .mode-fas-emblem { width: 20px; height: 20px; display: block; object-fit: contain; flex-shrink: 0; filter: drop-shadow(0 1px 3px #125cff20); }
.mode-trigger:hover { background: var(--fas-hover); }
.mode-caret { font-size: 15px; color: var(--ai-muted); }
.composer-footnote { text-align: center; color: var(--ai-muted); font-size: 10px; line-height: 1.5; margin: 9px auto 0; }
.selected-mode { margin: 12px 16px 0; display: flex; align-items: center; gap: 9px; padding: 9px 12px; background: var(--fas-accent-soft); border-radius: 12px; color: var(--fas-accent-text); }
.selected-mode > svg { width: 16px; height: 16px; }
.selected-mode strong { font-size: 11px; }
.selected-mode small { font-size: 10px; color: var(--ai-muted); }
.selected-mode button { margin-left: auto; border: 0; background: transparent; color: var(--ai-muted); }
.selected-mode button svg { width: 17px; height: 17px; }
.pending-files { display: flex; gap: 9px; padding: 12px 14px 0; overflow-x: auto; }
.pending-file { flex: 0 0 192px; width: 192px; height: 67px; display: flex; position: relative; align-items: center; gap: 9px; border: 1px solid var(--fas-border); border-radius: 12px; background: var(--fas-elevated); padding: 7px; }
.pending-file img, .pending-file video { width: 49px; height: 49px; object-fit: cover; border-radius: 8px; }
.pending-name { min-width: 0; font-size: 10px; white-space: nowrap; text-overflow: ellipsis; overflow: hidden; }
.pending-name small { display: block; padding-top: 5px; color: var(--ai-muted); }
.pending-file button { position: absolute; top: -5px; right: -5px; display: grid; place-items: center; width: 22px; height: 22px; background: var(--fas-card); border: 1px solid var(--fas-border-strong); border-radius: 50%; color: var(--fas-text); }
.pending-file button svg { width: 13px; height: 13px; }
.attach-menu { position: absolute; left: -7px; bottom: calc(100% + 17px); width: min(330px, calc(100vw - 48px)); max-height: min(450px, 65dvh); overflow-y: auto; padding: 8px; border-radius: 19px; border: 1px solid var(--fas-border-strong); box-shadow: 0 20px 50px var(--fas-shadow); background: var(--fas-card); z-index: 30; }
.attach-menu button { display: flex; align-items: center; gap: 12px; width: 100%; min-height: 40px; padding: 9px 11px; text-align: left; border: none; border-radius: 11px; background: transparent; color: var(--fas-text-soft); font-size: 11.5px !important; }
.attach-menu button:hover { background: var(--fas-hover); }
.attach-menu button svg { width: 18px; height: 18px; color: var(--fas-accent-text); flex: 0 0 auto; }
.attach-menu button small { color: var(--ai-muted); font-size: 10px; margin-left: auto; }
.menu-label { font-size: 9px; font-weight: 800; letter-spacing: .1em; color: var(--ai-muted); padding: 11px 10px 5px; }
.menu-divider { height: 1px; background: var(--fas-border); margin: 6px 0; }
.file-input { display: none; }
.ai-notice { position: absolute; z-index: 42; top: 72px; left: 50%; transform: translateX(-50%); max-width: min(90%, 600px); padding: 12px 15px; text-align: center; color: var(--fas-text); border: 1px solid var(--fas-accent-border); border-radius: 12px; background: var(--fas-card); font-size: 11px; line-height: 1.6; box-shadow: var(--fas-shadow-soft); }
.drop-overlay { position: absolute; z-index: 41; inset: 69px 20px 18px; border-radius: 19px; border: 2px dashed var(--fas-accent); background: color-mix(in srgb, var(--fas-bg) 94%, transparent); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; pointer-events: auto; }
.drop-overlay svg { width: 36px; height: 36px; color: var(--fas-accent-text); }
.drop-overlay strong { font-size: 17px; }
.drop-overlay span { font-size: 12px; color: var(--ai-muted); }
.mobile-scrim { display: none; }
@keyframes intro-appear { from { opacity: 0; transform: translateY(12px); } to { opacity: 1; transform: translateY(0); } }
@keyframes pulse-mic { 50% { box-shadow: 0 0 0 8px var(--fas-danger-soft); } }
@media (max-width: 900px) {
  .fas-ai-workspace, .fas-ai-workspace.history-collapsed { grid-template-columns: minmax(0,1fr); }
  .ai-history { position: absolute; left: 0; top: 0; bottom: 0; z-index: 50; width: min(285px, 86vw); transform: translateX(-105%); opacity: 0; border-right: 1px solid var(--fas-border); box-shadow: 15px 0 40px var(--fas-shadow); transition: transform .25s ease, opacity .25s ease; }
  .ai-history.is-mobile-open { transform: translateX(0); opacity: 1; pointer-events: auto; }
  .history-collapsed .ai-history { pointer-events: none; }
  .history-collapsed .ai-history.is-mobile-open { pointer-events: auto; }
  .mobile-scrim { display: block; position: absolute; inset: 0; z-index: 49; background: #0007; }
  .chat-scroll { padding-inline: 22px; }
  .composer-area { padding-inline: 18px; }
  .head-role { display: none; }
}
@media (max-width: 570px) {
  .fas-ai-workspace { height: calc(100dvh - 83px); min-height: 380px; }
  .chat-header { height: 57px; min-height: 57px; padding-inline: 10px; }
  .chat-scroll { padding: 8px 14px 15px; }
  .welcome-content { padding: 10px 0; }
  .welcome-mark { width: min(213px, 65vw); height: 97px; margin-bottom: 12px; }
  .welcome-brand { font-size: 9px; margin-bottom: 11px; }
  .welcome-content h1 { font-size: clamp(24px, 7vw, 37px); }
  .welcome-subtitle { font-size: 11px; margin-top: 10px; }
  .welcome-prompts { grid-template-columns: 1fr; gap: 7px; margin-top: 19px; }
  .welcome-prompts button { min-height: 43px; padding: 8px 11px; font-size: 10.5px !important; }
  .prompt-icon { width: 27px; height: 27px; border-radius: 9px; }
  .composer-area { padding: 0 10px 10px; }
  .composer-wrap { border-radius: 22px; }
  .composer-form textarea { min-height: 53px; padding: 14px 16px 6px; }
  .composer-controls { padding: 4px 8px 9px; }
  .composer-footnote { font-size: 9px; }
  .message-user .message-body { max-width: 91%; }
  .attach-menu { bottom: calc(100% + 11px); }
}
@media (prefers-reduced-motion: reduce) {
  .fas-ai-workspace *, .fas-ai-workspace *::before, .fas-ai-workspace *::after { transition: none !important; animation: none !important; scroll-behavior: auto !important; }
}
</style>
