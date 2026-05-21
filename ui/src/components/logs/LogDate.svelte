<script>
    import tooltip from "@/actions/tooltip";
    import CommonHelper from "@/utils/CommonHelper";

    export let date;

    const tooltipData = {
        // generate the tooltip text as getter to speed up the initial load
        // in case the component is used with large number of items
        get text() {
            return date.replace("Z", " UTC");
        },
    };
</script>

<span class="txt-nowrap" use:tooltip={tooltipData}>
    {CommonHelper.formatToLocalDate(date, "yyyy-MM-dd HH:mm:ss.SSS")}
</span>
