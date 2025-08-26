import { useState } from "react";
import { User, Settings as SettingsIcon, Key, Bot } from "lucide-react";
import {
	Tabs,
	TabsContent,
	TabsList,
	TabsTrigger,
} from "../components/ui/tabs";
import { Header } from "../components/Header";
import { ProfileSettings } from "../components/ProfileSettings";
import { AccountSettings } from "../components/AccountSettings";
import { APIKeySettings } from "../components/APIKeySettings";
import { LLMSettings } from "../components/LLMSettings";

export function Settings() {
	const [activeTab, setActiveTab] = useState("profiles");

	// Dummy function for file select (Settings page doesn't upload files)
	const handleFileSelect = () => {
		// No file upload in settings
	};

	return (
		<div className="min-h-screen bg-gray-50 dark:bg-gray-900">
			<div className="mx-auto w-full max-w-6xl px-2 sm:px-6 md:px-8 py-3 sm:py-6">
				{/* Use the same Header component as Homepage */}
				<Header onFileSelect={handleFileSelect} />

				{/* Main Content Container with same styling as Homepage */}
				<div className="bg-white dark:bg-gray-800 rounded-xl p-2 sm:p-6 mt-4 sm:mt-6">
					<div className="mb-4 sm:mb-8">
						<h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-2">
							Settings
						</h1>
						<p className="text-gray-600 dark:text-gray-400">
							Manage your account settings and transcription profiles.
						</p>
					</div>

					{/* Tabbed Interface */}
						<Tabs
							value={activeTab}
							onValueChange={setActiveTab}
							className="space-y-4 sm:space-y-6"
						>
							<TabsList className="grid w-full grid-cols-4 items-center h-auto bg-gray-100 dark:bg-gray-800 p-1 rounded-xl">
                            <TabsTrigger
                                value="profiles"
                                aria-label="Profiles"
                                className="flex items-center justify-center gap-2 h-10 py-2 data-[state=active]:bg-white dark:data-[state=active]:bg-gray-700 data-[state=active]:text-gray-900 dark:data-[state=active]:text-gray-100 text-gray-600 dark:text-gray-400 font-medium rounded-lg text-xs sm:text-sm"
                            >
									<SettingsIcon className="h-4 w-4" />
									<span className="hidden sm:inline">Profiles</span>
								</TabsTrigger>
                            <TabsTrigger
                                value="account"
                                aria-label="Account"
                                className="flex items-center justify-center gap-2 h-10 py-2 data-[state=active]:bg-white dark:data-[state=active]:bg-gray-700 data-[state=active]:text-gray-900 dark:data-[state=active]:text-gray-100 text-gray-600 dark:text-gray-400 font-medium rounded-lg text-xs sm:text-sm"
                            >
									<User className="h-4 w-4" />
									<span className="hidden sm:inline">Account</span>
								</TabsTrigger>
                            <TabsTrigger
                                value="apikeys"
                                aria-label="API Keys"
                                className="flex items-center justify-center gap-2 h-10 py-2 data-[state=active]:bg-white dark:data-[state=active]:bg-gray-700 data-[state=active]:text-gray-900 dark:data-[state=active]:text-gray-100 text-gray-600 dark:text-gray-400 font-medium rounded-lg text-xs sm:text-sm"
                            >
									<Key className="h-4 w-4" />
									<span className="hidden sm:inline">API Keys</span>
								</TabsTrigger>
                            <TabsTrigger
                                value="llms"
                                aria-label="LLMs"
                                className="flex items-center justify-center gap-2 h-10 py-2 data-[state=active]:bg-white dark:data-[state=active]:bg-gray-700 data-[state=active]:text-gray-900 dark:data-[state=active]:text-gray-100 text-gray-600 dark:text-gray-400 font-medium rounded-lg text-xs sm:text-sm"
                            >
									<Bot className="h-4 w-4" />
									<span className="hidden sm:inline">LLMs</span>
								</TabsTrigger>
							</TabsList>

						{/* Profiles Tab */}
						<TabsContent value="profiles" className="space-y-6">
							<ProfileSettings />
						</TabsContent>

						{/* Account Tab */}
						<TabsContent value="account" className="space-y-6">
							<AccountSettings />
						</TabsContent>

						{/* API Keys Tab */}
						<TabsContent value="apikeys" className="space-y-6">
							<APIKeySettings />
						</TabsContent>

						{/* LLMs Tab */}
						<TabsContent value="llms" className="space-y-6">
							<LLMSettings />
						</TabsContent>
					</Tabs>
				</div>
			</div>
		</div>
	);
}
