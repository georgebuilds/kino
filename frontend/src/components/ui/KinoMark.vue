<!--
  Kino — bull elk silhouette in dark green, facing right.
  Rendered from a raster asset; size is the bounding box edge.
-->
<template>
  <img
    :src="elkUrl"
    :width="size"
    :height="size"
    alt="Kino"
    class="kino-mark"
    draggable="false"
  />
</template>

<script setup lang="ts">
import elkUrl from '../../assets/images/elk.png'

withDefaults(defineProps<{ size?: number }>(), { size: 32 })
</script>

<style scoped>
.kino-mark {
  display: block;
  object-fit: contain;
  user-select: none;
}

/* lift the dark-green silhouette against dark surfaces */
[data-theme='dark'] .kino-mark {
  filter: brightness(1.25);
}
</style>
