<script>
    import Toggler from "@/components/base/Toggler.svelte";

    export let label = "More options";
</script>

<div tabindex="0" role="button" aria-label={label} class="btn btn-sm btn-circle btn-transparent flex-gap-0">
    <i class="ri-more-line" aria-hidden="true" />
    <Toggler class="dropdown dropdown-right dropdown-nowrap">
        <slot />
    </Toggler>
</div>
