<template>
  <Teleport to="body">
    <div class="modal-backdrop" @click.self="$emit('close')">
      <div class="modal" role="dialog" :aria-label="isEdit ? 'Edit category' : 'New category'">

        <div class="modal__header">
          <h2 class="modal__title">{{ isEdit ? 'Edit category' : 'New category' }}</h2>
          <button class="btn btn--ghost btn--icon" @click="$emit('close')" aria-label="Close">
            <X :size="16" />
          </button>
        </div>

        <div class="modal__body">

          <!-- Name -->
          <div class="field">
            <label class="field__label" for="cat-name">Name</label>
            <input
              id="cat-name"
              ref="nameInput"
              v-model="draft.name"
              class="field__input"
              placeholder="e.g. Groceries"
              maxlength="64"
              required
              @keydown.enter="save"
            />
          </div>

          <!-- Parent -->
          <div class="field">
            <label class="field__label" for="cat-parent">Parent category</label>
            <select id="cat-parent" v-model="parentIdVal" class="field__input field__select" @change="onParentChange">
              <option :value="null">None (top level)</option>
              <optgroup v-if="rootParents.length" label="Top level">
                <option v-for="p in rootParents" :key="p.id" :value="p.id">
                  {{ p.name }}
                </option>
              </optgroup>
              <optgroup v-if="childParents.length" label="Second level">
                <option v-for="p in childParents" :key="p.id" :value="p.id">
                  {{ indentedName(p) }}
                </option>
              </optgroup>
            </select>
            <p v-if="depthWarning" class="field__hint field__hint--warn">{{ depthWarning }}</p>
          </div>

          <!-- Color -->
          <div class="field">
            <label class="field__label">Color</label>
            <div class="color-swatches">
              <button
                v-for="swatch in COLOR_SWATCHES"
                :key="swatch"
                type="button"
                class="color-swatch"
                :class="{ 'color-swatch--active': draft.color === swatch }"
                :style="{ background: swatch }"
                :title="swatch"
                @click="draft.color = swatch"
              >
                <Check v-if="draft.color === swatch" :size="10" />
              </button>
              <!-- Custom hex -->
              <label class="color-swatch color-swatch--custom" :style="{ background: isCustomColor ? draft.color : 'transparent' }" title="Custom colour">
                <input
                  type="color"
                  class="color-swatch__native"
                  :value="draft.color"
                  @input="onCustomColor"
                />
                <Pipette v-if="!isCustomColor" :size="12" style="color: var(--color-text-tertiary)" />
                <Check v-else :size="10" style="color: #fff" />
              </label>
            </div>
          </div>

          <!-- Icon -->
          <div class="field">
            <label class="field__label">Icon</label>
            <div class="icon-grid">
              <button
                v-for="opt in ICON_OPTIONS"
                :key="opt.name"
                type="button"
                class="icon-btn"
                :class="{ 'icon-btn--active': draft.icon === opt.name }"
                :title="opt.name"
                @click="draft.icon = opt.name"
              >
                <component :is="opt.component" :size="16" />
              </button>
            </div>
          </div>

          <!-- Income toggle — only for top-level categories -->
          <div v-if="parentIdVal == null" class="field field--inline">
            <label class="field__label" for="cat-income" style="margin-bottom:0">Income category</label>
            <div class="toggle-wrap">
              <input id="cat-income" v-model="draft.isIncome" type="checkbox" class="toggle-input" />
              <label for="cat-income" class="toggle-label" />
            </div>
            <p class="field__hint" style="grid-column: 1/-1">
              Enable for salary, freelance, interest and other money-in sources.
            </p>
          </div>

          <p v-if="saveError" class="modal__error">{{ saveError }}</p>
        </div>

        <div class="modal__footer">
          <button class="btn btn--ghost" @click="$emit('close')">Cancel</button>
          <button class="btn btn--primary" :disabled="!canSave || saving" @click="save">
            <Loader2 v-if="saving" :size="14" class="spin" />
            {{ isEdit ? 'Save changes' : 'Create' }}
          </button>
        </div>

      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { X, Check, Pipette, Loader2 } from 'lucide-vue-next'
import { useCategoriesStore } from '../../stores/categories'
import type { Category } from '../../stores/categories'
import { ICON_OPTIONS, COLOR_SWATCHES } from '../../utils/categoryIcons'

const props = defineProps<{
  category?:       Category   // undefined = create mode
  defaultParentId?: number    // pre-select a parent (e.g. from "Add subcategory")
}>()

const emit = defineEmits<{
  (e: 'close'):           void
  (e: 'saved', cat: Category): void
}>()

const catStore = useCategoriesStore()

// ── Draft ─────────────────────────────────────────────────────────────────────

const isEdit = computed(() => props.category != null)

function makeDraft(): Omit<Category, 'id' | 'isSystem'> {
  if (props.category) {
    return {
      name:      props.category.name,
      parentId:  props.category.parentId,
      color:     props.category.color || '#5A6B60',
      icon:      props.category.icon  || 'tag',
      isIncome:  props.category.isIncome,
      sortOrder: props.category.sortOrder,
    }
  }
  // New category — inherit parent's income flag if parent is set
  const parent = props.defaultParentId != null
    ? catStore.findById(props.defaultParentId)
    : null
  return {
    name:      '',
    parentId:  props.defaultParentId ?? undefined,
    color:     parent?.color ?? '#5A6B60',
    icon:      parent?.icon  ?? 'tag',
    isIncome:  parent?.isIncome ?? false,
    sortOrder: 0,
  }
}

const draft = ref(makeDraft())
const parentIdVal = ref<number | null>(draft.value.parentId ?? null)

// Keep draft.parentId in sync with the select
function onParentChange() {
  draft.value.parentId = parentIdVal.value ?? undefined
  // Inherit income flag from parent
  if (parentIdVal.value != null) {
    const parent = catStore.findById(parentIdVal.value)
    if (parent) draft.value.isIncome = parent.isIncome
  }
}

// ── Parent options ────────────────────────────────────────────────────────────

const eligible = computed(() => catStore.eligibleParents(props.category?.id))

const rootParents  = computed(() => eligible.value.filter(c => !c.parentId))
const childParents = computed(() => eligible.value.filter(c =>  c.parentId))

function indentedName(cat: Category): string {
  if (!cat.parentId) return cat.name
  const parent = catStore.findById(cat.parentId)
  return parent ? `${parent.name} › ${cat.name}` : cat.name
}

const depthWarning = computed(() => {
  if (parentIdVal.value == null) return null
  const d = catStore.depth(parentIdVal.value)
  if (d >= 1) return 'This will create a third-level category (the maximum depth).'
  return null
})

// ── Color ─────────────────────────────────────────────────────────────────────

const isCustomColor = computed(() =>
  !!draft.value.color && !COLOR_SWATCHES.includes(draft.value.color)
)

function onCustomColor(e: Event) {
  draft.value.color = (e.target as HTMLInputElement).value
}

// ── Save / error ──────────────────────────────────────────────────────────────

const saving    = ref(false)
const saveError = ref<string | null>(null)

const canSave = computed(() => draft.value.name.trim().length > 0)

async function save() {
  if (!canSave.value || saving.value) return
  saving.value    = true
  saveError.value = null
  try {
    const payload = {
      ...draft.value,
      name:     draft.value.name.trim(),
      parentId: parentIdVal.value ?? undefined,
    }
    let saved: Category
    if (isEdit.value && props.category) {
      const full: Category = {
        ...props.category,
        ...payload,
        isSystem: props.category.isSystem,
      }
      await catStore.update(full)
      saved = full
    } else {
      saved = await catStore.create(payload)
    }
    emit('saved', saved)
    emit('close')
  } catch (e: any) {
    saveError.value = e?.message ?? 'Failed to save'
  } finally {
    saving.value = false
  }
}

// ── Focus ─────────────────────────────────────────────────────────────────────

const nameInput = ref<HTMLInputElement | null>(null)
onMounted(() => nextTick(() => nameInput.value?.focus()))
</script>

<style scoped>
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
  width: 100%; max-width: 480px;
  box-shadow: 0 24px 64px rgba(0,0,0,0.35);
  animation: slide-up var(--duration-base) var(--ease-out);
  max-height: 90vh; overflow-y: auto;
}
@keyframes slide-up {
  from { opacity: 0; transform: translateY(10px) }
  to   { opacity: 1; transform: translateY(0) }
}

.modal__header {
  display: flex; align-items: center; justify-content: space-between;
  padding: var(--space-5) var(--space-5) 0;
  position: sticky; top: 0; z-index: 1;
  background: var(--color-surface-1);
  border-radius: var(--radius-lg) var(--radius-lg) 0 0;
}
.modal__title { font-size: 17px; font-weight: 600; color: var(--color-text-primary); }
.btn--icon { width: 32px; height: 32px; padding: 0; display: flex; align-items: center; justify-content: center; }

.modal__body {
  padding: var(--space-5);
  display: flex; flex-direction: column; gap: var(--space-5);
}
.modal__footer {
  display: flex; justify-content: flex-end; gap: var(--space-3);
  padding: var(--space-4) var(--space-5);
  border-top: 1px solid var(--color-border);
  position: sticky; bottom: 0;
  background: var(--color-surface-1);
  border-radius: 0 0 var(--radius-lg) var(--radius-lg);
}
.modal__error {
  background: rgba(220,38,38,0.08); border: 1px solid rgba(220,38,38,0.25);
  border-radius: var(--radius-md); padding: var(--space-3);
  font: var(--text-body-sm); color: var(--color-expense);
}

/* Fields */
.field { display: flex; flex-direction: column; gap: var(--space-2); }
.field--inline { display: grid; grid-template-columns: 1fr auto; align-items: center; gap: var(--space-2) var(--space-4); }
.field__label { font: var(--text-body-sm); font-weight: 600; color: var(--color-text-secondary); }
.field__input {
  height: 38px; padding: 0 var(--space-3);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-primary);
  font: var(--text-body); outline: none;
  width: 100%;
}
.field__input:focus { border-color: var(--color-primary); box-shadow: 0 0 0 3px rgba(26,138,97,0.15); }
.field__select {
  appearance: none; cursor: pointer;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='11' height='11' viewBox='0 0 24 24' fill='none' stroke='%235A6B60' stroke-width='2.5' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
  background-repeat: no-repeat; background-position: right var(--space-3) center;
  padding-right: 28px;
}
.field__hint {
  font: var(--text-body-sm); color: var(--color-text-tertiary); margin-top: -var(--space-1);
}
.field__hint--warn { color: #C4943A; }

/* Color swatches */
.color-swatches {
  display: flex; flex-wrap: wrap; gap: var(--space-2);
}
.color-swatch {
  width: 28px; height: 28px; border-radius: var(--radius-sm);
  border: 2px solid transparent;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; flex-shrink: 0;
  transition: transform var(--duration-fast) var(--ease-out), border-color var(--duration-fast) var(--ease-out);
}
.color-swatch:hover { transform: scale(1.12); }
.color-swatch--active { border-color: var(--color-text-primary); }
.color-swatch--active svg { color: #fff; }

.color-swatch--custom {
  background: var(--color-surface-2) !important;
  border: 2px dashed var(--color-border);
  cursor: pointer;
  position: relative; overflow: hidden;
}
.color-swatch--custom:hover { border-color: var(--color-text-tertiary); }
.color-swatch__native {
  position: absolute; inset: 0; opacity: 0; cursor: pointer; width: 100%; height: 100%;
}

/* Icon grid */
.icon-grid {
  display: grid; grid-template-columns: repeat(10, 1fr); gap: 4px;
}
.icon-btn {
  width: 36px; height: 36px;
  display: flex; align-items: center; justify-content: center;
  border-radius: var(--radius-sm);
  border: 1.5px solid transparent;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition:
    background var(--duration-fast) var(--ease-out),
    border-color var(--duration-fast) var(--ease-out),
    color var(--duration-fast) var(--ease-out);
}
.icon-btn:hover {
  background: var(--color-surface-2);
  color: var(--color-text-primary);
}
.icon-btn--active {
  background: var(--color-surface-2);
  border-color: var(--color-primary);
  color: var(--color-primary);
}

/* Toggle switch */
.toggle-wrap { position: relative; width: 40px; height: 22px; flex-shrink: 0; }
.toggle-input { position: absolute; opacity: 0; width: 0; height: 0; }
.toggle-label {
  display: block; width: 40px; height: 22px;
  background: var(--color-border); border-radius: 11px;
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-out);
}
.toggle-label::after {
  content: ''; position: absolute;
  top: 3px; left: 3px;
  width: 16px; height: 16px; border-radius: 50%;
  background: #fff;
  transition: transform var(--duration-fast) var(--ease-out);
  box-shadow: 0 1px 3px rgba(0,0,0,0.3);
}
.toggle-input:checked + .toggle-label { background: var(--color-primary); }
.toggle-input:checked + .toggle-label::after { transform: translateX(18px); }

.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg) } }
</style>
