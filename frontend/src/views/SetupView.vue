<template>
  <div class="setup">
    <div class="setup__card card card--lg">
      <!-- Mark + wordmark -->
      <div class="setup__brand">
        <KinoMark :size="48" />
        <span class="setup__wordmark">kino</span>
      </div>

      <h1 class="text-display-sm setup__headline">
        Your finances, on your terms.
      </h1>
      <p class="text-body setup__sub">
        All data stays in a single file you own. Store it anywhere —
        we recommend a cloud-synced folder so you always have a backup.
      </p>

      <!-- Cloud folder suggestions -->
      <div v-if="cloudFolders.length" class="setup__clouds">
        <p class="text-label" style="color: var(--color-text-tertiary); margin-bottom: var(--space-2);">
          DETECTED ON YOUR MACHINE
        </p>
        <div class="setup__cloud-list">
          <div
            v-for="folder in cloudFolders"
            :key="folder.path"
            class="setup__cloud-item"
            :class="{ 'setup__cloud-item--selected': selectedFolder === folder.path }"
            tabindex="0"
            role="option"
            :aria-selected="selectedFolder === folder.path"
            @click="selectedFolder = folder.path"
            @keydown.enter="selectedFolder = folder.path"
            @keydown.space.prevent="selectedFolder = folder.path"
          >
            <div class="setup__cloud-icon">
              <Cloud :size="16" />
            </div>
            <span class="text-body" style="font-weight: 500;">{{ folder.name }}</span>
            <Check v-if="selectedFolder === folder.path" :size="14" class="setup__cloud-check" />
          </div>
        </div>
      </div>

      <div class="divider" />

      <!-- Actions -->
      <div class="setup__actions">
        <button class="btn btn--primary" style="flex: 1;" @click="onCreate">
          <FilePlus :size="16" />
          Create new file
        </button>
        <button class="btn btn--ghost" style="flex: 1;" @click="onOpen">
          <FolderOpen :size="16" />
          Open existing
        </button>
      </div>

      <p v-if="error" class="text-body-sm setup__error">{{ error }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { FilePlus, FolderOpen, Cloud, Check } from 'lucide-vue-next'
import { CreateFile, OpenFile, CloudFolderSuggestions } from '../../wailsjs/go/main/App'
import KinoMark from '../components/ui/KinoMark.vue'

const router = useRouter()
const cloudFolders = ref<{ name: string; path: string }[]>([])
const selectedFolder = ref('')
const error = ref('')

onMounted(async () => {
  try {
    cloudFolders.value = await CloudFolderSuggestions()
    if (cloudFolders.value.length) {
      selectedFolder.value = cloudFolders.value[0].path
    }
  } catch {
    // Non-fatal — user can still pick manually
  }
})

async function onCreate() {
  error.value = ''
  try {
    // TODO: pass selectedFolder.value once CreateFile() backend accepts a path parameter
    await CreateFile()
    router.push('/')
  } catch (e: any) {
    error.value = e?.message ?? 'Could not create file.'
  }
}

async function onOpen() {
  error.value = ''
  try {
    await OpenFile()
    router.push('/')
  } catch (e: any) {
    error.value = e?.message ?? 'Could not open file.'
  }
}
</script>

<style scoped>
.setup {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-8);
  background: var(--color-bg);
  min-height: 100vh;
}

.setup__card {
  width: 100%;
  max-width: 460px;
}

.setup__brand {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-8);
}

.setup__wordmark {
  font: var(--text-display-sm);
  font-weight: 700;
  letter-spacing: 1px;
  color: var(--color-text-primary);
}

.setup__headline {
  margin-bottom: var(--space-3);
  letter-spacing: -0.3px;
}

.setup__sub {
  color: var(--color-text-secondary);
  margin-bottom: var(--space-6);
  line-height: 1.6;
}

.setup__clouds {
  margin-bottom: var(--space-6);
}

.setup__cloud-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.setup__cloud-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-3);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  cursor: pointer;
  transition:
    border-color var(--duration-fast) var(--ease-out),
    background-color var(--duration-fast) var(--ease-out);
}

.setup__cloud-item:hover {
  background: var(--color-surface-2);
}

.setup__cloud-item--selected {
  border-color: var(--color-primary);
  background: var(--color-primary-50);
}

[data-theme="dark"] .setup__cloud-item--selected {
  background: rgba(26, 138, 97, 0.10);
}

.setup__cloud-icon {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-2);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
}

.setup__cloud-check {
  margin-left: auto;
  color: var(--color-primary);
}

.setup__actions {
  display: flex;
  gap: var(--space-3);
}

.setup__error {
  margin-top: var(--space-4);
  color: var(--color-expense);
  text-align: center;
}
</style>
