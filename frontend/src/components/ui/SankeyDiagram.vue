<template>
  <div ref="containerEl" class="sankey-wrap">
    <svg
      v-if="width > 0 && hasData"
      :width="width"
      :height="height"
      class="sankey-svg"
      role="img"
      aria-label="Cash flow diagram"
    >
      <title>Cash flow diagram</title>
      <defs>
        <!-- Per-link gradients -->
        <linearGradient
          v-for="link in linkLayouts"
          :key="`grad-${link.sourceId}-${link.targetId}`"
          :id="`grad-${link.sourceId}-${link.targetId}`"
          gradientUnits="userSpaceOnUse"
          :x1="nodeWidth + labelW"
          :y1="0"
          :x2="rightX"
          :y2="0"
        >
          <stop offset="0%"   :stop-color="link.color" stop-opacity="0.55" />
          <stop offset="100%" :stop-color="link.targetColor" stop-opacity="0.30" />
        </linearGradient>
      </defs>

      <!-- Link ribbons (drawn first, behind nodes) -->
      <path
        v-for="link in linkLayouts"
        :key="`link-${link.sourceId}-${link.targetId}`"
        :d="linkPath(link)"
        :fill="`url(#grad-${link.sourceId}-${link.targetId})`"
        class="sankey-link"
      />

      <!-- Left nodes (income) -->
      <g v-for="node in leftLayouts" :key="node.id">
        <rect
          :x="labelW"
          :y="node.y"
          :width="nodeWidth"
          :height="node.height"
          :fill="node.color"
          rx="3"
          class="sankey-node"
        />
        <!-- Label: left of node -->
        <text
          :x="labelW - 10"
          :y="node.y + node.height / 2"
          class="sankey-label sankey-label--left"
        >
          {{ node.name }}
        </text>
        <!-- Amount below name -->
        <text
          :x="labelW - 10"
          :y="node.y + node.height / 2 + 14"
          class="sankey-amount sankey-label--left"
        >
          {{ formatMoney(node.valueCents) }}
        </text>
      </g>

      <!-- Right nodes (expenses + saved) -->
      <g v-for="node in rightLayouts" :key="node.id">
        <rect
          :x="rightX"
          :y="node.y"
          :width="nodeWidth"
          :height="node.height"
          :fill="node.color"
          rx="3"
          class="sankey-node"
        />
        <!-- Label: right of node -->
        <text
          :x="rightX + nodeWidth + 10"
          :y="node.y + node.height / 2"
          class="sankey-label sankey-label--right"
        >
          {{ node.name }}
        </text>
        <text
          :x="rightX + nodeWidth + 10"
          :y="node.y + node.height / 2 + 14"
          class="sankey-amount sankey-label--right"
        >
          {{ formatMoney(node.valueCents) }}
        </text>
      </g>
    </svg>

    <!-- Empty / no-data state -->
    <div v-else-if="width > 0" class="sankey-empty">
      <p>No cash flow data for this period.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

// ── Props ─────────────────────────────────────────────────────────────────────
interface SNode {
  id:         string
  name:       string
  color:      string
  valueCents: number
}

interface SLink {
  sourceId:   string
  targetId:   string
  valueCents: number
  color:      string   // source color
}

const props = defineProps<{
  leftNodes:  SNode[]
  rightNodes: SNode[]
  links:      SLink[]
}>()

// ── Sizing ────────────────────────────────────────────────────────────────────
const containerEl = ref<HTMLDivElement | null>(null)
const width  = ref(0)

// Adaptive height: more nodes → taller chart, capped at 560
const height = computed(() => {
  const n = Math.max(props.leftNodes.length, props.rightNodes.length)
  return Math.min(Math.max(n * 72 + 60, 260), 560)
})

let ro: ResizeObserver | null = null
onMounted(() => {
  ro = new ResizeObserver(entries => {
    width.value = entries[0].contentRect.width
  })
  if (containerEl.value) {
    ro.observe(containerEl.value)
    width.value = containerEl.value.getBoundingClientRect().width
  }
})
onUnmounted(() => ro?.disconnect())

// ── Layout constants ──────────────────────────────────────────────────────────
const nodeWidth = 14
const labelW    = 148   // pixels reserved for left labels
const vPad      = 16    // top/bottom padding
const nodeGap   = 8     // gap between nodes in the same column

// ── Derived ───────────────────────────────────────────────────────────────────
const hasData = computed(() =>
  props.leftNodes.length > 0 || props.rightNodes.length > 0
)

const rightX = computed(() => width.value - labelW - nodeWidth)

// ── Layout computation ────────────────────────────────────────────────────────

interface NodeLayout extends SNode {
  x:      number
  y:      number
  height: number
}

interface LinkLayout {
  sourceId:    string
  targetId:    string
  color:       string
  targetColor: string
  valueCents:  number
  // source side band
  sy0: number
  sy1: number
  // target side band
  ty0: number
  ty1: number
}

function layoutColumn(nodes: SNode[], x: number, totalValue: number, availH: number): NodeLayout[] {
  if (nodes.length === 0) return []
  const totalGaps   = nodeGap * (nodes.length - 1)
  const nodeH_total = availH - totalGaps

  let y = vPad
  return nodes.map(n => {
    const h = totalValue > 0
      ? Math.max((n.valueCents / totalValue) * nodeH_total, 4)
      : 4
    const layout: NodeLayout = { ...n, x, y, height: h }
    y += h + nodeGap
    return layout
  })
}

const leftLayouts = computed<NodeLayout[]>(() => {
  if (!width.value) return []
  const totalIncome = props.leftNodes.reduce((s, n) => s + n.valueCents, 0)
  const availH = height.value - 2 * vPad - nodeGap * (props.leftNodes.length - 1)
  return layoutColumn(props.leftNodes, labelW, totalIncome, availH + nodeGap * (props.leftNodes.length - 1))
})

const rightLayouts = computed<NodeLayout[]>(() => {
  if (!width.value) return []
  const totalRight = props.rightNodes.reduce((s, n) => s + n.valueCents, 0)
  const availH = height.value - 2 * vPad - nodeGap * (props.rightNodes.length - 1)
  return layoutColumn(props.rightNodes, rightX.value, totalRight, availH + nodeGap * (props.rightNodes.length - 1))
})

const linkLayouts = computed<LinkLayout[]>(() => {
  if (!width.value) return []

  const leftByID       = new Map(leftLayouts.value.map(n  => [n.id, n]))
  const rightByID      = new Map(rightLayouts.value.map(n => [n.id, n]))
  const rightColorByID = new Map(props.rightNodes.map(n   => [n.id, n.color]))

  // Per-node fill pointers: where the next link band starts within this node
  const srcFill = new Map(leftLayouts.value.map(n  => [n.id, n.y]))
  const tgtFill = new Map(rightLayouts.value.map(n => [n.id, n.y]))

  const layouts: LinkLayout[] = []

  for (const link of props.links) {
    const src = leftByID.get(link.sourceId)
    const tgt = rightByID.get(link.targetId)
    if (!src || !tgt || link.valueCents <= 0) continue

    // Band height is proportional to the link's share of each node's value,
    // scaled by that node's rendered pixel height. The ribbon naturally
    // "widens" or "narrows" as it flows left → right.
    const srcH = src.valueCents > 0
      ? (link.valueCents / src.valueCents) * src.height : 0
    const tgtH = tgt.valueCents > 0
      ? (link.valueCents / tgt.valueCents) * tgt.height : 0

    const sy0 = srcFill.get(link.sourceId)!
    const ty0 = tgtFill.get(link.targetId)!

    srcFill.set(link.sourceId, sy0 + srcH)
    tgtFill.set(link.targetId, ty0 + tgtH)

    layouts.push({
      sourceId:    link.sourceId,
      targetId:    link.targetId,
      color:       link.color,
      targetColor: rightColorByID.get(link.targetId) ?? link.color,
      valueCents:  link.valueCents,
      sy0, sy1: sy0 + srcH,
      ty0, ty1: ty0 + tgtH,
    })
  }
  return layouts
})

// ── SVG path ─────────────────────────────────────────────────────────────────
function linkPath(l: LinkLayout): string {
  const x0 = labelW + nodeWidth
  const x1 = rightX.value
  const mx = (x0 + x1) / 2

  return [
    `M ${x0} ${l.sy0}`,
    `C ${mx} ${l.sy0}, ${mx} ${l.ty0}, ${x1} ${l.ty0}`,
    `L ${x1} ${l.ty1}`,
    `C ${mx} ${l.ty1}, ${mx} ${l.sy1}, ${x0} ${l.sy1}`,
    'Z',
  ].join(' ')
}

// ── Formatting ────────────────────────────────────────────────────────────────
function formatMoney(cents: number) {
  const abs = Math.abs(cents)
  if (abs >= 100000) return (cents < 0 ? '-$' : '$') + (abs / 100 / 1000).toFixed(1) + 'k'
  return (cents < 0 ? '-$' : '$') +
    (abs / 100).toLocaleString('en-US', { minimumFractionDigits: 0, maximumFractionDigits: 0 })
}
</script>

<style scoped>
.sankey-wrap {
  width: 100%;
  min-height: 260px;
  position: relative;
}

.sankey-svg {
  display: block;
  overflow: visible;
}

.sankey-node {
  transition: opacity var(--duration-fast) var(--ease-out);
}
.sankey-node:hover { opacity: 0.85; }

.sankey-link {
  transition: opacity var(--duration-fast) var(--ease-out);
}
.sankey-link:hover { opacity: 0.8; }

.sankey-label {
  font-family: var(--font-ui, system-ui);
  font-size: 12px;
  font-weight: 600;
  fill: var(--color-text-primary, #e8f0ec);
  dominant-baseline: middle;
}

.sankey-label--left  { text-anchor: end; }
.sankey-label--right { text-anchor: start; }

.sankey-amount {
  font-family: var(--font-ui, system-ui);
  font-size: 11px;
  font-weight: 400;
  fill: var(--color-text-tertiary, #5A6B60);
  dominant-baseline: middle;
}

.sankey-empty {
  display: flex; align-items: center; justify-content: center;
  min-height: 260px;
  font: var(--text-body);
  color: var(--color-text-tertiary);
}
</style>
