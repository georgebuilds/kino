import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  ListCategories,
  CreateCategory,
  UpdateCategory,
  DeleteCategory,
  GetCategoryTransactionCount,
} from '../../wailsjs/go/main/App'
import type { models } from '../../wailsjs/go/models'

export type Category = models.Category

// ── Tree types ────────────────────────────────────────────────────────────────

export interface CategoryNode {
  cat:      Category
  depth:    number          // 0 = root, 1 = child, 2 = grandchild (max)
  children: CategoryNode[]  // only populated for depth 0 and 1
}

export const MAX_DEPTH = 2  // 0-indexed → 3 visual levels

// ── Store ─────────────────────────────────────────────────────────────────────

export const useCategoriesStore = defineStore('categories', () => {
  const categories  = ref<Category[]>([])
  const loading     = ref(false)
  const error       = ref<string | null>(null)
  const initialised = ref(false)

  // ── Tree ──────────────────────────────────────────────────────────────────

  /**
   * Flat list → 3-level tree.
   * Root nodes (parentId == null) are at depth 0.
   * Their children at depth 1, grandchildren at depth 2.
   * Any deeper nodes are silently attached at depth 2 (shouldn't happen).
   */
  const tree = computed<CategoryNode[]>(() => {
    const byId  = new Map<number, Category>(categories.value.map(c => [c.id, c]))
    const roots: CategoryNode[] = []
    const nodeById = new Map<number, CategoryNode>()

    // Build node objects — two passes: roots first, then children
    for (const cat of categories.value) {
      const node: CategoryNode = { cat, depth: 0, children: [] }
      nodeById.set(cat.id, node)
    }

    for (const cat of categories.value) {
      const node = nodeById.get(cat.id)!
      if (cat.parentId == null) {
        node.depth = 0
        roots.push(node)
      } else {
        const parent = nodeById.get(cat.parentId)
        if (parent) {
          node.depth = Math.min(parent.depth + 1, MAX_DEPTH)
          parent.children.push(node)
        } else {
          // orphan — treat as root
          node.depth = 0
          roots.push(node)
        }
      }
    }

    // Sort by sort_order then id at each level
    const sort = (nodes: CategoryNode[]) => {
      nodes.sort((a, b) => a.cat.sortOrder - b.cat.sortOrder || a.cat.id - b.cat.id)
      nodes.forEach(n => sort(n.children))
    }
    sort(roots)
    void byId  // silence unused-var lint
    return roots
  })

  /** Depth of a category by id (0 = root, 1 = child, 2 = grandchild). */
  function depth(id: number): number {
    const find = (nodes: CategoryNode[]): number => {
      for (const n of nodes) {
        if (n.cat.id === id) return n.depth
        const d = find(n.children)
        if (d !== -1) return d
      }
      return -1
    }
    return Math.max(0, find(tree.value))
  }

  /**
   * Categories that are legal parents for a new/edited category.
   * A parent must be at depth < MAX_DEPTH (so the child stays within limit).
   * Pass excludeId when editing — exclude the category itself and all descendants.
   */
  function eligibleParents(excludeId?: number): Category[] {
    const excluded = new Set<number>()
    if (excludeId != null) {
      // Collect the subtree rooted at excludeId
      const collect = (nodes: CategoryNode[]) => {
        for (const n of nodes) {
          if (n.cat.id === excludeId || excluded.has(n.cat.id)) {
            excluded.add(n.cat.id)
            n.children.forEach(c => collect([c]))
          } else {
            collect(n.children)
          }
        }
      }
      collect(tree.value)
    }

    return categories.value.filter(c =>
      !excluded.has(c.id) &&
      depth(c.id) < MAX_DEPTH
    )
  }

  // ── Derived ───────────────────────────────────────────────────────────────

  /** Top-level categories only */
  const topLevel = computed(() => categories.value.filter(c => !c.parentId))

  /** All expense categories */
  const expenseCategories = computed(() =>
    categories.value.filter(c => !c.isIncome && c.name !== 'Transfers')
  )

  /** All income categories */
  const incomeCategories = computed(() =>
    categories.value.filter(c => c.isIncome)
  )

  /** Quick lookup by id */
  function findById(id: number): Category | null {
    return categories.value.find(c => c.id === id) ?? null
  }

  // ── Actions ───────────────────────────────────────────────────────────────

  async function fetch() {
    if (loading.value) return
    loading.value = true
    error.value   = null
    try {
      categories.value = await ListCategories()
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to load categories'
    } finally {
      loading.value     = false
      initialised.value = true
    }
  }

  async function create(draft: Omit<Category, 'id' | 'isSystem'>) {
    const created = await CreateCategory(draft as Category)
    categories.value.push(created)
    categories.value = [...categories.value] // trigger reactivity
    return created
  }

  async function update(cat: Category) {
    await UpdateCategory(cat)
    const idx = categories.value.findIndex(c => c.id === cat.id)
    if (idx !== -1) categories.value[idx] = cat
    categories.value = [...categories.value]
  }

  async function remove(id: number) {
    await DeleteCategory(id)
    // Also detach any children (they'll become roots; backend sets parent_id=NULL)
    categories.value = categories.value
      .filter(c => c.id !== id)
      .map(c => c.parentId === id ? { ...c, parentId: undefined } : c)
  }

  async function getTransactionCount(id: number): Promise<number> {
    return Number(await GetCategoryTransactionCount(id))
  }

  return {
    categories, loading, error, initialised,
    tree,
    topLevel, expenseCategories, incomeCategories,
    fetch, create, update, remove, findById,
    depth, eligibleParents, getTransactionCount,
  }
})
