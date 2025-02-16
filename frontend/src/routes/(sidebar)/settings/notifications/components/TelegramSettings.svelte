<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import { Input, Label, Toggle, Button } from 'flowbite-svelte';

    export let settings: {
        enabled: boolean;
        token: string;
        chat_ids: number[];
    };

    let newChatId = '';

    const dispatch = createEventDispatcher();

    function handleChange() {
        dispatch('change', settings);
    }

    function addChatId() {
        if (newChatId && !isNaN(Number(newChatId))) {
            settings.chat_ids = [...settings.chat_ids, Number(newChatId)];
            newChatId = '';
            handleChange();
        }
    }

    function removeChatId(chatId: number) {
        settings.chat_ids = settings.chat_ids.filter(id => id !== chatId);
        handleChange();
    }
</script>

<div class="space-y-4">
    <div class="flex items-center justify-between">
        <Label>Enable Telegram Integration</Label>
        <Toggle 
            bind:checked={settings.enabled} 
            on:change={handleChange}
        />
    </div>

    {#if settings.enabled}
        <div>
            <Label for="token">Bot Token</Label>
            <Input
                id="token"
                type="password"
                placeholder="Bot token from @BotFather"
                bind:value={settings.token}
                on:change={handleChange}
            />
            <p class="text-sm text-gray-500 mt-1">
                Create a bot and get the token from 
                <a 
                    href="https://t.me/botfather" 
                    target="_blank" 
                    rel="noopener noreferrer"
                    class="text-blue-600 hover:underline"
                >
                    @BotFather
                </a>
            </p>
        </div>

        <div>
            <Label>Chat IDs</Label>
            <div class="flex gap-2 mb-2">
                <Input
                    type="text"
                    placeholder="Enter chat ID"
                    bind:value={newChatId}
                />
                <Button 
                    size="sm"
                    on:click={addChatId}
                >
                    Add
                </Button>
            </div>
            <p class="text-sm text-gray-500 mb-2">
                Start a chat with your bot and use @get_id_bot to find your chat ID
            </p>

            {#if settings.chat_ids.length > 0}
                <div class="flex flex-wrap gap-2">
                    {#each settings.chat_ids as chatId}
                        <div class="flex items-center gap-2 bg-gray-100 dark:bg-gray-700 px-3 py-1 rounded-full">
                            <span class="text-sm">{chatId}</span>
                            <button
                                class="text-red-500 hover:text-red-700"
                                on:click={() => removeChatId(chatId)}
                            >
                                ×
                            </button>
                        </div>
                    {/each}
                </div>
            {:else}
                <p class="text-sm text-gray-500">No chat IDs added yet</p>
            {/if}
        </div>
    {/if}
</div> 