export namespace core {
	
	export class ApplyResult {
	    Locale: string;
	    LocaleName: string;
	    BackupPath: string;
	    CleanBackup: string;
	    WasRunning: boolean;
	    Forced: boolean;
	    Relaunched: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ApplyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Locale = source["Locale"];
	        this.LocaleName = source["LocaleName"];
	        this.BackupPath = source["BackupPath"];
	        this.CleanBackup = source["CleanBackup"];
	        this.WasRunning = source["WasRunning"];
	        this.Forced = source["Forced"];
	        this.Relaunched = source["Relaunched"];
	    }
	}
	export class LocaleInfo {
	    code: string;
	    name: string;
	    englishName: string;
	    translations: number;
	    coverage: number;
	
	    static createFrom(source: any = {}) {
	        return new LocaleInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.name = source["name"];
	        this.englishName = source["englishName"];
	        this.translations = source["translations"];
	        this.coverage = source["coverage"];
	    }
	}
	export class RestoreResult {
	    AlreadyEnglish: boolean;
	    BackupPath: string;
	    WasRunning: boolean;
	    Forced: boolean;
	    Relaunched: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RestoreResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.AlreadyEnglish = source["AlreadyEnglish"];
	        this.BackupPath = source["BackupPath"];
	        this.WasRunning = source["WasRunning"];
	        this.Forced = source["Forced"];
	        this.Relaunched = source["Relaunched"];
	    }
	}
	export class Status {
	    AppDir: string;
	    ResourcesDir: string;
	    AsarPath: string;
	    CleanBackupPath: string;
	    StatePath: string;
	    AppVersion: string;
	    CurrentLanguage: string;
	    LocaleName: string;
	    HasCleanBackup: boolean;
	    CleanBackupStatus: string;
	    LastPatchedAt?: string;
	    StateIsStale: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.AppDir = source["AppDir"];
	        this.ResourcesDir = source["ResourcesDir"];
	        this.AsarPath = source["AsarPath"];
	        this.CleanBackupPath = source["CleanBackupPath"];
	        this.StatePath = source["StatePath"];
	        this.AppVersion = source["AppVersion"];
	        this.CurrentLanguage = source["CurrentLanguage"];
	        this.LocaleName = source["LocaleName"];
	        this.HasCleanBackup = source["HasCleanBackup"];
	        this.CleanBackupStatus = source["CleanBackupStatus"];
	        this.LastPatchedAt = source["LastPatchedAt"];
	        this.StateIsStale = source["StateIsStale"];
	    }
	}

}

