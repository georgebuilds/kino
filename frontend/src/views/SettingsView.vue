<template>
  <div class="page">
    <div class="page-header">
      <h1 class="page-title">Settings</h1>
      <p class="page-subtitle">Preferences and account setup</p>
    </div>

    <!-- ── Narrow section: appearance + data file ─────────────────────────── -->
    <div class="settings-narrow">

      <!-- Appearance -->
      <div class="card">
        <h2 class="text-heading" style="margin-bottom: var(--space-5);">Appearance</h2>
        <div class="settings-row">
          <div>
            <p class="text-body" style="font-weight: 500;">Theme</p>
            <p class="text-body-sm" style="color: var(--color-text-secondary); margin-top: 2px;">
              Matches your system by default
            </p>
          </div>
          <div class="theme-toggle">
            <button
              v-for="opt in themeOptions"
              :key="opt.value"
              class="theme-toggle__btn"
              :class="{ 'theme-toggle__btn--active': preference === opt.value }"
              @click="setPreference(opt.value)"
            >
              <component :is="opt.icon" :size="14" />
              {{ opt.label }}
            </button>
          </div>
        </div>
      </div>

      <!-- Updates -->
      <div class="card">
        <h2 class="text-heading" style="margin-bottom: var(--space-5);">About</h2>
        <div class="settings-row">
          <div>
            <p class="text-body" style="font-weight: 500;">Version</p>
            <p class="text-body-sm" style="color: var(--color-text-secondary); margin-top: 2px;">
              {{ appVersion }}
            </p>
          </div>
          <button
            class="btn btn--ghost btn--sm"
            :disabled="updateStatus === 'checking' || updateStatus === 'updating'"
            @click="checkForUpdate"
          >
            <Loader2 v-if="updateStatus === 'checking'" :size="14" class="spin" />
            <RefreshCw v-else :size="14" />
            Check for updates
          </button>
        </div>

        <!-- Update available / installing -->
        <div v-if="updateStatus === 'available' || updateStatus === 'updating'" class="update-banner update-banner--available">
          <ArrowUpCircle :size="15" style="flex-shrink:0;" />
          <div style="flex:1; min-width:0;">
            <span class="text-body-sm" style="font-weight:500;">
              v{{ updateInfo!.version }} is available
            </span>
            <template v-if="updateStatus === 'updating'">
              <div class="update-progress">
                <div class="update-progress__bar">
                  <div class="update-progress__fill" :style="{ width: `${downloadProgress}%` }"></div>
                </div>
                <span class="update-progress__label">
                  {{ downloadProgress > 0 ? `${downloadProgress}%` : 'Downloading…' }}
                </span>
              </div>
            </template>
          </div>
          <button
            class="btn btn--primary btn--sm"
            :disabled="updateStatus === 'updating'"
            @click="applyUpdate"
          >
            <Loader2 v-if="updateStatus === 'updating'" :size="14" class="spin" />
            <Download v-else :size="14" />
            Install
          </button>
        </div>

        <!-- Up to date -->
        <div v-if="updateStatus === 'up-to-date'" class="update-banner update-banner--ok">
          <CheckCircle :size="15" style="flex-shrink:0;" />
          <p class="text-body-sm">You're up to date.</p>
        </div>

        <!-- Error -->
        <div v-if="updateStatus === 'error'" class="update-banner update-banner--error">
          <p class="text-body-sm">{{ updateError }}</p>
        </div>
      </div>

      <!-- Data file -->
      <div class="card">
        <h2 class="text-heading" style="margin-bottom: var(--space-5);">Data file</h2>
        <div class="settings-row" style="margin-bottom: var(--space-4);">
          <div style="min-width: 0;">
            <p class="text-body" style="font-weight: 500;">Current file</p>
            <p class="text-body-sm truncate" style="color: var(--color-text-secondary); margin-top: 2px; max-width: 320px;">
              {{ filePath || 'No file open' }}
            </p>
          </div>
          <button class="btn btn--ghost btn--sm" @click="onMove">
            <FolderInput :size="14" />
            Move
          </button>
        </div>
        <div v-if="cloudFolders.length" class="settings__cloud-tip">
          <Cloud :size="14" style="flex-shrink:0;" />
          <p class="text-body-sm">
            <strong>Tip:</strong> move your .kino file into
            {{ cloudFolders.map(f => f.name).join(' or ') }}
            for automatic backup across devices.
          </p>
        </div>
      </div>

    </div>

    <!-- ── Categories ─────────────────────────────────────────────────────── -->
    <div class="settings-wide">
      <div class="card">

        <div class="cat-section-header">
          <div>
            <h2 class="text-heading">Categories</h2>
            <p class="text-body-sm" style="color: var(--color-text-secondary); margin-top: 2px;">
              Up to 3 levels deep. System categories can't be deleted but you can add subcategories under them.
            </p>
          </div>
          <button class="btn btn--primary btn--sm" @click="openCreate()">
            <Plus :size="14" />
            Add category
          </button>
        </div>

        <div v-if="catStore.loading" class="cat-skeleton">
          <div v-for="i in 6" :key="i" class="skeleton-row" />
        </div>

        <div v-else class="cat-tree">
          <template v-for="root in catStore.tree" :key="root.cat.id">

            <!-- ── Root (depth 0) ─────────────────────────────────────────── -->
            <div class="cat-row cat-row--root" :class="{ 'cat-row--income': root.cat.isIncome }">
              <span class="cat-dot" :style="{ background: root.cat.color }" />
              <component :is="iconComponent(root.cat.icon)" :size="14" class="cat-icon" />
              <span class="cat-name">{{ root.cat.name }}</span>
              <span v-if="root.cat.isIncome" class="cat-badge cat-badge--income">income</span>
              <span v-if="root.cat.isSystem" class="cat-badge cat-badge--system">
                <Lock :size="10" /> system
              </span>
              <div class="cat-actions">
                <button
                  v-if="root.depth < 2"
                  class="btn btn--ghost btn--xs cat-action-btn"
                  title="Add subcategory"
                  @click="openCreate(root.cat.id)"
                >
                  <Plus :size="11" /> Sub
                </button>
                <button
                  v-if="!root.cat.isSystem"
                  class="btn btn--ghost btn--icon-xs"
                  title="Edit"
                  @click="openEdit(root.cat)"
                >
                  <Pencil :size="12" />
                </button>
                <button
                  v-if="!root.cat.isSystem"
                  class="btn btn--ghost btn--icon-xs btn--danger"
                  title="Delete"
                  @click="confirmDelete(root.cat)"
                >
                  <Trash2 :size="12" />
                </button>
              </div>
            </div>

            <!-- ── Depth 1 children ───────────────────────────────────────── -->
            <template v-for="child in root.children" :key="child.cat.id">
              <div class="cat-row cat-row--child">
                <span class="cat-indent" />
                <span class="cat-dot" :style="{ background: child.cat.color }" />
                <component :is="iconComponent(child.cat.icon)" :size="13" class="cat-icon" />
                <span class="cat-name">{{ child.cat.name }}</span>
                <span v-if="child.cat.isSystem" class="cat-badge cat-badge--system">
                  <Lock :size="10" /> system
                </span>
                <div class="cat-actions">
                  <button
                    v-if="child.depth < 2"
                    class="btn btn--ghost btn--xs cat-action-btn"
                    title="Add subcategory"
                    @click="openCreate(child.cat.id)"
                  >
                    <Plus :size="11" /> Sub
                  </button>
                  <button
                    v-if="!child.cat.isSystem"
                    class="btn btn--ghost btn--icon-xs"
                    title="Edit"
                    @click="openEdit(child.cat)"
                  >
                    <Pencil :size="12" />
                  </button>
                  <button
                    v-if="!child.cat.isSystem"
                    class="btn btn--ghost btn--icon-xs btn--danger"
                    title="Delete"
                    @click="confirmDelete(child.cat)"
                  >
                    <Trash2 :size="12" />
                  </button>
                </div>
              </div>

              <!-- ── Depth 2 grandchildren ──────────────────────────────── -->
              <div
                v-for="gc in child.children"
                :key="gc.cat.id"
                class="cat-row cat-row--grandchild"
              >
                <span class="cat-indent" />
                <span class="cat-indent" />
                <span class="cat-dot cat-dot--sm" :style="{ background: gc.cat.color }" />
                <component :is="iconComponent(gc.cat.icon)" :size="12" class="cat-icon cat-icon--sm" />
                <span class="cat-name cat-name--sm">{{ gc.cat.name }}</span>
                <div class="cat-actions">
                  <button
                    v-if="!gc.cat.isSystem"
                    class="btn btn--ghost btn--icon-xs"
                    title="Edit"
                    @click="openEdit(gc.cat)"
                  >
                    <Pencil :size="12" />
                  </button>
                  <button
                    v-if="!gc.cat.isSystem"
                    class="btn btn--ghost btn--icon-xs btn--danger"
                    title="Delete"
                    @click="confirmDelete(gc.cat)"
                  >
                    <Trash2 :size="12" />
                  </button>
                </div>
              </div>
            </template>

          </template>
        </div>

      </div>
    </div>

    <!-- ── Delete confirm ─────────────────────────────────────────────────── -->
    <Teleport to="body">
      <div v-if="deleteTarget" class="modal-backdrop" @click.self="deleteTarget = null">
        <div class="modal modal--sm" role="alertdialog" aria-labelledby="settings-delete-title">
          <div class="modal__header">
            <h2 id="settings-delete-title" class="modal__title">Delete "{{ deleteTarget.name }}"?</h2>
          </div>
          <div class="modal__body">
            <p class="text-body" style="color: var(--color-text-secondary);">
              <span v-if="deleteTxCount > 0">
                <strong style="color:var(--color-text-primary)">{{ deleteTxCount }}</strong>
                {{ deleteTxCount === 1 ? 'transaction' : 'transactions' }} will be moved to
                <em>Uncategorized</em>.
              </span>
              <span v-else>No transactions use this category.</span>
              <span v-if="deleteChildCount > 0">
                {{ ' ' }}<strong style="color:var(--color-text-primary)">{{ deleteChildCount }}</strong>
                {{ deleteChildCount === 1 ? 'subcategory' : 'subcategories' }} will become top-level.
              </span>
            </p>
            <p v-if="deleteError" class="modal__error">{{ deleteError }}</p>
            <div class="modal__footer" style="border:none; padding: 0; margin-top: var(--space-2)">
              <button class="btn btn--ghost" @click="deleteTarget = null">Cancel</button>
              <button class="btn btn--danger" :disabled="deleteBusy" @click="doDelete">
                <Loader2 v-if="deleteBusy" :size="14" class="spin" />
                Delete
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- ── Category modal ─────────────────────────────────────────────────── -->
    <CategoryModal
      v-if="showModal"
      :category="editTarget ?? undefined"
      :default-parent-id="defaultParentId ?? undefined"
      @close="closeModal"
      @saved="closeModal"
    />

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, markRaw } from 'vue'
import {
  Monitor, Sun, Moon, Cloud, FolderInput,
  Plus, Pencil, Trash2, Loader2, Lock,
  RefreshCw, ArrowUpCircle, CheckCircle, Download,
} from 'lucide-vue-next'
import { useThemeStore } from '../stores/theme'
import type { ThemePreference } from '../stores/theme'
import { storeToRefs } from 'pinia'
import { GetFileState, MoveFile, CloudFolderSuggestions, GetAppVersion, CheckForUpdate, ApplyUpdate } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import type { main } from '../../wailsjs/go/models'
import { useCategoriesStore } from '../stores/categories'
import type { Category } from '../stores/categories'
import { iconComponent } from '../utils/categoryIcons'
import CategoryModal from '../components/ui/CategoryModal.vue'

// ── Theme ─────────────────────────────────────────────────────────────────────
const store = useThemeStore()
const { preference } = storeToRefs(store)
const { setPreference } = store

const themeOptions: { value: ThemePreference; label: string; icon: any }[] = [
  { value: 'system', label: 'System', icon: markRaw(Monitor) },
  { value: 'light',  label: 'Light',  icon: markRaw(Sun) },
  { value: 'dark',   label: 'Dark',   icon: markRaw(Moon) },
]

// ── Updates ───────────────────────────────────────────────────────────────────
const appVersion  = ref('')
type UpdateStatus = 'idle' | 'checking' | 'up-to-date' | 'available' | 'updating' | 'error'
const updateStatus   = ref<UpdateStatus>('idle')
const updateInfo     = ref<main.UpdateInfo | null>(null)
const updateError    = ref<string | null>(null)
const downloadProgress = ref(0)

async function checkForUpdate() {
  updateStatus.value = 'checking'
  updateError.value  = null
  try {
    const info = await CheckForUpdate()
    if (info.available) {
      updateInfo.value   = info
      updateStatus.value = 'available'
    } else {
      updateStatus.value = 'up-to-date'
    }
  } catch (e: any) {
    updateError.value  = e?.message ?? 'Failed to check for updates'
    updateStatus.value = 'error'
  }
}

async function applyUpdate() {
  updateStatus.value   = 'updating'
  updateError.value    = null
  downloadProgress.value = 0

  const offProgress = EventsOn('update:progress', (data: { percent: number }) => {
    downloadProgress.value = Math.round(data.percent)
  })

  try {
    await ApplyUpdate()
    // On Linux/Windows the app quits; on macOS it relaunches — either way
    // we won't reach this line in normal flow.
  } catch (e: any) {
    updateError.value  = e?.message ?? 'Update failed'
    updateStatus.value = 'error'
  } finally {
    offProgress()
  }
}

// ── Data file ─────────────────────────────────────────────────────────────────
const filePath    = ref('')
const cloudFolders = ref<{ name: string; path: string }[]>([])

onMounted(async () => {
  try {
    const [state, folders, ver] = await Promise.all([
      GetFileState(),
      CloudFolderSuggestions(),
      GetAppVersion(),
    ])
    filePath.value     = state.path
    cloudFolders.value = folders
    appVersion.value   = ver
  } catch (e: any) {
    console.error('Failed to load settings:', e?.message ?? e)
  }
})

async function onMove() {
  try {
    const state = await MoveFile()
    if (state?.path) filePath.value = state.path
  } catch (e: any) {
    console.error('Failed to move file:', e?.message ?? e)
  }
}

// ── Categories ────────────────────────────────────────────────────────────────
const catStore = useCategoriesStore()

// Modal state
const showModal       = ref(false)
const editTarget      = ref<Category | null>(null)
const defaultParentId = ref<number | null>(null)

function openCreate(parentId?: number) {
  editTarget.value      = null
  defaultParentId.value = parentId ?? null
  showModal.value       = true
}

function openEdit(cat: Category) {
  editTarget.value      = cat
  defaultParentId.value = null
  showModal.value       = true
}

function closeModal() {
  showModal.value       = false
  editTarget.value      = null
  defaultParentId.value = null
}

// Delete state
const deleteTarget   = ref<Category | null>(null)
const deleteTxCount  = ref(0)
const deleteChildCount = ref(0)
const deleteBusy     = ref(false)
const deleteError    = ref<string | null>(null)

async function confirmDelete(cat: Category) {
  deleteError.value = null
  const [txCount] = await Promise.all([
    catStore.getTransactionCount(cat.id),
  ])
  deleteTxCount.value    = txCount
  deleteChildCount.value = catStore.categories.filter(c => c.parentId === cat.id).length
  deleteTarget.value = cat
}

async function doDelete() {
  if (!deleteTarget.value) return
  deleteBusy.value  = true
  deleteError.value = null
  try {
    await catStore.remove(deleteTarget.value.id)
    deleteTarget.value = null
  } catch (e: any) {
    deleteError.value = e?.message ?? 'Failed to delete'
  } finally {
    deleteBusy.value = false
  }
}
</script>

<style scoped>
/* ── Layout ── */
.settings-narrow {
  max-width: 560px;
  display: flex; flex-direction: column; gap: var(--space-5);
}
.settings-wide {
  max-width: 720px;
  margin-top: var(--space-5);
}

/* ── Existing settings styles ── */
.settings-row {
  display: flex; align-items: center; justify-content: space-between; gap: var(--space-4);
}
.theme-toggle {
  display: flex; background: var(--color-surface-2);
  border-radius: var(--radius-md); padding: 3px; gap: 2px;
}
.theme-toggle__btn {
  display: inline-flex; align-items: center; gap: var(--space-1);
  padding: var(--space-1) var(--space-3);
  border-radius: calc(var(--radius-md) - 2px);
  border: none; background: transparent; color: var(--color-text-secondary);
  font: var(--text-body-sm); font-weight: 500; cursor: pointer;
  transition: background var(--duration-fast) var(--ease-out),
              color       var(--duration-fast) var(--ease-out),
              box-shadow  var(--duration-fast) var(--ease-out);
  white-space: nowrap;
}
.theme-toggle__btn--active {
  background: var(--color-surface-1); color: var(--color-text-primary);
  box-shadow: var(--shadow-sm);
}
.settings__cloud-tip {
  display: flex; align-items: flex-start; gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  background: var(--color-surface-2); border-radius: var(--radius-md);
  color: var(--color-text-secondary);
}

/* ── Update banners ── */
.update-banner {
  display: flex; align-items: center; gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  margin-top: var(--space-4);
}
.update-banner--available {
  background: rgba(26,138,97,0.08);
  border: 1px solid rgba(26,138,97,0.2);
  color: var(--color-text-primary);
}
.update-banner--ok {
  background: var(--color-surface-2);
  color: var(--color-text-secondary);
}
.update-banner--error {
  background: rgba(220,38,38,0.08);
  border: 1px solid rgba(220,38,38,0.2);
  color: var(--color-expense);
}
.update-progress {
  display: flex; align-items: center; gap: var(--space-2);
  margin-top: 6px;
}
.update-progress__bar {
  flex: 1; height: 4px; border-radius: 2px;
  background: rgba(26,138,97,0.15); overflow: hidden;
}
.update-progress__fill {
  height: 100%; border-radius: 2px;
  background: #1A8A61; transition: width 0.2s ease;
}
.update-progress__label {
  font-size: 11px; color: var(--color-text-secondary);
  white-space: nowrap;
}

/* ── Category section ── */
.cat-section-header {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: var(--space-4); margin-bottom: var(--space-5);
}

/* ── Tree rows ── */
.cat-tree { display: flex; flex-direction: column; }

.cat-row {
  display: flex; align-items: center; gap: var(--space-2);
  padding: var(--space-2) var(--space-1);
  border-radius: var(--radius-sm);
  min-height: 36px;
  transition: background var(--duration-fast) var(--ease-out);
}
.cat-row:hover { background: var(--color-surface-2); }

.cat-row--root { font-weight: 500; }
.cat-row--child { padding-left: var(--space-1); }
.cat-row--grandchild { font-size: 13px; color: var(--color-text-secondary); }

.cat-indent { width: 20px; flex-shrink: 0; }

.cat-dot {
  width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0;
}
.cat-dot--sm { width: 8px; height: 8px; }

.cat-icon { color: var(--color-text-secondary); flex-shrink: 0; }
.cat-icon--sm { opacity: 0.7; }

.cat-name { flex: 1; min-width: 0; truncate: true; }
.cat-name--sm { color: var(--color-text-secondary); }

/* Badges */
.cat-badge {
  display: inline-flex; align-items: center; gap: 3px;
  font-size: 10px; font-weight: 600; letter-spacing: 0.04em; text-transform: uppercase;
  border-radius: var(--radius-sm);
  padding: 1px 6px;
  flex-shrink: 0;
}
.cat-badge--income {
  color: var(--color-primary);
  background: rgba(26,138,97,0.12);
}
.cat-badge--system {
  color: var(--color-text-tertiary);
  background: var(--color-surface-2);
}

/* Row actions — hidden until hover */
.cat-actions {
  display: flex; align-items: center; gap: 2px;
  opacity: 0;
  transition: opacity var(--duration-fast) var(--ease-out);
  flex-shrink: 0;
}
.cat-row:hover .cat-actions { opacity: 1; }

.cat-action-btn {
  height: 24px; padding: 0 var(--space-2);
  font-size: 11px; display: flex; align-items: center; gap: 3px;
}
.btn--icon-xs {
  width: 26px; height: 26px; padding: 0;
  display: flex; align-items: center; justify-content: center;
}
.btn--danger { color: var(--color-expense); }
.btn--danger:hover { background: rgba(220,38,38,0.08); }
.btn--xs { font: var(--text-body-sm); }

/* ── Skeleton ── */
.cat-skeleton { display: flex; flex-direction: column; gap: var(--space-2); margin-top: var(--space-2); }
.skeleton-row {
  height: 36px; border-radius: var(--radius-sm);
  background: linear-gradient(90deg, var(--color-surface-2) 25%, var(--color-surface-1) 50%, var(--color-surface-2) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
@keyframes shimmer { to { background-position: -200% 0 } }

/* ── Delete modal ── */
.modal-backdrop {
  position: fixed; inset: 0; z-index: 300;
  background: rgba(0,0,0,0.5); backdrop-filter: blur(4px);
  display: flex; align-items: center; justify-content: center;
  padding: var(--space-4);
  animation: fade-in var(--duration-base) var(--ease-out);
}
@keyframes fade-in { from { opacity: 0 } to { opacity: 1 } }

.modal {
  background: var(--color-surface-1);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  width: 100%; max-width: 420px;
  box-shadow: 0 24px 64px rgba(0,0,0,0.3);
  animation: slide-up var(--duration-base) var(--ease-out);
}
@keyframes slide-up {
  from { opacity: 0; transform: translateY(10px) }
  to   { opacity: 1; transform: translateY(0) }
}
.modal--sm { max-width: 380px; }
.modal__header {
  display: flex; align-items: center;
  padding: var(--space-5) var(--space-5) 0;
}
.modal__title { font-size: 17px; font-weight: 600; color: var(--color-text-primary); }
.modal__body {
  padding: var(--space-5);
  display: flex; flex-direction: column; gap: var(--space-4);
}
.modal__footer {
  display: flex; justify-content: flex-end; gap: var(--space-3);
}
.modal__error {
  background: rgba(220,38,38,0.08); border: 1px solid rgba(220,38,38,0.25);
  border-radius: var(--radius-md); padding: var(--space-3);
  font: var(--text-body-sm); color: var(--color-expense);
}
.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg) } }
</style>
