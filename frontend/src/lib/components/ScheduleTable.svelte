<script lang="ts">
	import { DAYS, fmtTime, lookup } from '$lib/format';
	import { buildTimetable } from '$lib/schedule';
	import type { Rec } from '$lib/api/types';

	// Klasik haftalık ders tablosu: satır = ders saati, sütun = gün.
	let {
		rows,
		courses,
		classes = new Map<string, string>(),
		showClass = false
	}: { rows: Rec[]; courses: Map<string, string>; classes?: Map<string, string>; showClass?: boolean } = $props();

	const table = $derived(buildTimetable(rows));
</script>

{#if table.slots.length}
	<div class="overflow-x-auto rounded-lg border border-zinc-200 bg-white shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
		<table class="w-full min-w-[720px] border-collapse text-sm">
			<caption class="sr-only">Haftalık ders programı</caption>
			<thead class="bg-zinc-50 text-left text-xs font-semibold tracking-wide text-zinc-600 uppercase dark:bg-zinc-800/60 dark:text-zinc-300">
				<tr>
					<th scope="col" class="w-32 border-b border-zinc-200 px-3 py-2.5 dark:border-zinc-800">Saat</th>
					{#each table.days as day (day)}
						<th scope="col" class="border-b border-l border-zinc-200 px-3 py-2.5 dark:border-zinc-800">{DAYS[day]}</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each table.slots as slot (slot.start)}
					<tr class="border-b border-zinc-200 last:border-b-0 dark:border-zinc-800">
						<th scope="row" class="bg-zinc-50 px-3 py-3 text-left font-semibold whitespace-nowrap tabular-nums dark:bg-zinc-800/40">
							{fmtTime(slot.start)} – {fmtTime(slot.end)}
						</th>
						{#each table.days as day (day)}
							<td class="border-l border-zinc-200 px-2 py-2 align-top dark:border-zinc-800">
								{#each table.at(day, slot.start) as lesson (lesson.key)}
									<div class="mb-1 rounded-md border border-brand-100 bg-brand-50 px-2.5 py-1.5 last:mb-0 dark:border-brand-900 dark:bg-brand-950/50">
										<p class="font-semibold text-brand-900 dark:text-brand-100">{lookup(courses, lesson.courseId)}</p>
										{#if showClass}<p class="text-xs text-zinc-600 dark:text-zinc-400">{lookup(classes, lesson.classId)}</p>{/if}
									</div>
								{/each}
							</td>
						{/each}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
