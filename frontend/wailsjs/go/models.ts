export namespace db {
	
	export class NetWorthPoint {
	    month: string;
	    netWorth: number;
	    assets: number;
	    liabilities: number;
	
	    static createFrom(source: any = {}) {
	        return new NetWorthPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.month = source["month"];
	        this.netWorth = source["netWorth"];
	        this.assets = source["assets"];
	        this.liabilities = source["liabilities"];
	    }
	}
	export class PossibleDupe {
	    newTx: models.Transaction;
	    existingTx: models.Transaction;
	
	    static createFrom(source: any = {}) {
	        return new PossibleDupe(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.newTx = this.convertValues(source["newTx"], models.Transaction);
	        this.existingTx = this.convertValues(source["existingTx"], models.Transaction);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TxFilter {
	    accountId?: number;
	    categoryId?: number;
	    dateFrom: string;
	    dateTo: string;
	    search: string;
	    limit: number;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new TxFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accountId = source["accountId"];
	        this.categoryId = source["categoryId"];
	        this.dateFrom = source["dateFrom"];
	        this.dateTo = source["dateTo"];
	        this.search = source["search"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }
	}
	export class TxPage {
	    transactions: models.Transaction[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new TxPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.transactions = this.convertValues(source["transactions"], models.Transaction);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class BudgetLine {
	    id: number;
	    categoryId: number;
	    categoryName: string;
	    categoryColor: string;
	    categoryIcon: string;
	    budgetCents: number;
	    spentCents: number;
	    period: string;
	    rollsOver: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BudgetLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.categoryId = source["categoryId"];
	        this.categoryName = source["categoryName"];
	        this.categoryColor = source["categoryColor"];
	        this.categoryIcon = source["categoryIcon"];
	        this.budgetCents = source["budgetCents"];
	        this.spentCents = source["spentCents"];
	        this.period = source["period"];
	        this.rollsOver = source["rollsOver"];
	    }
	}
	export class UnbudgetedLine {
	    categoryId: number;
	    categoryName: string;
	    categoryColor: string;
	    categoryIcon: string;
	    spentCents: number;
	
	    static createFrom(source: any = {}) {
	        return new UnbudgetedLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.categoryId = source["categoryId"];
	        this.categoryName = source["categoryName"];
	        this.categoryColor = source["categoryColor"];
	        this.categoryIcon = source["categoryIcon"];
	        this.spentCents = source["spentCents"];
	    }
	}
	export class BudgetPage {
	    lines: BudgetLine[];
	    unbudgeted: UnbudgetedLine[];
	    totalBudgetCents: number;
	    totalSpentCents: number;
	
	    static createFrom(source: any = {}) {
	        return new BudgetPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lines = this.convertValues(source["lines"], BudgetLine);
	        this.unbudgeted = this.convertValues(source["unbudgeted"], UnbudgetedLine);
	        this.totalBudgetCents = source["totalBudgetCents"];
	        this.totalSpentCents = source["totalSpentCents"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FlowLink {
	    sourceId: string;
	    targetId: string;
	    valueCents: number;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new FlowLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceId = source["sourceId"];
	        this.targetId = source["targetId"];
	        this.valueCents = source["valueCents"];
	        this.color = source["color"];
	    }
	}
	export class FlowNode {
	    id: string;
	    name: string;
	    color: string;
	    valueCents: number;
	    isIncome: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FlowNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.color = source["color"];
	        this.valueCents = source["valueCents"];
	        this.isIncome = source["isIncome"];
	    }
	}
	export class CashFlow {
	    leftNodes: FlowNode[];
	    rightNodes: FlowNode[];
	    links: FlowLink[];
	    incomeCents: number;
	    expenseCents: number;
	    savedCents: number;
	
	    static createFrom(source: any = {}) {
	        return new CashFlow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.leftNodes = this.convertValues(source["leftNodes"], FlowNode);
	        this.rightNodes = this.convertValues(source["rightNodes"], FlowNode);
	        this.links = this.convertValues(source["links"], FlowLink);
	        this.incomeCents = source["incomeCents"];
	        this.expenseCents = source["expenseCents"];
	        this.savedCents = source["savedCents"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CatTotal {
	    categoryId: number;
	    categoryName: string;
	    color: string;
	    amountCents: number;
	
	    static createFrom(source: any = {}) {
	        return new CatTotal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.categoryId = source["categoryId"];
	        this.categoryName = source["categoryName"];
	        this.color = source["color"];
	        this.amountCents = source["amountCents"];
	    }
	}
	export class CloudFolder {
	    name: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new CloudFolder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	    }
	}
	export class FileState {
	    path: string;
	    isOpen: boolean;
	    isNew: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.isOpen = source["isOpen"];
	        this.isNew = source["isNew"];
	    }
	}
	
	
	export class ImportResult {
	    inserted: number;
	    skipped: number;
	    possibleDupes: db.PossibleDupe[];
	    fileName: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inserted = source["inserted"];
	        this.skipped = source["skipped"];
	        this.possibleDupes = this.convertValues(source["possibleDupes"], db.PossibleDupe);
	        this.fileName = source["fileName"];
	        this.source = source["source"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MonthSummary {
	    netWorthCents: number;
	    netWorthDeltaCents: number;
	    incomeCents: number;
	    expenseCents: number;
	    savedCents: number;
	    topCategory: string;
	    topCategoryCents: number;
	    categoryTotals: CatTotal[];
	
	    static createFrom(source: any = {}) {
	        return new MonthSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.netWorthCents = source["netWorthCents"];
	        this.netWorthDeltaCents = source["netWorthDeltaCents"];
	        this.incomeCents = source["incomeCents"];
	        this.expenseCents = source["expenseCents"];
	        this.savedCents = source["savedCents"];
	        this.topCategory = source["topCategory"];
	        this.topCategoryCents = source["topCategoryCents"];
	        this.categoryTotals = this.convertValues(source["categoryTotals"], CatTotal);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace models {
	
	export class Account {
	    id: number;
	    name: string;
	    type: string;
	    institution: string;
	    balanceCents: number;
	    currency: string;
	    isHidden: boolean;
	    sortOrder: number;
	    // Go type: time
	    lastSyncedAt?: any;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Account(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.institution = source["institution"];
	        this.balanceCents = source["balanceCents"];
	        this.currency = source["currency"];
	        this.isHidden = source["isHidden"];
	        this.sortOrder = source["sortOrder"];
	        this.lastSyncedAt = this.convertValues(source["lastSyncedAt"], null);
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Budget {
	    id: number;
	    categoryId: number;
	    amountCents: number;
	    period: string;
	    rollsOver: boolean;
	    // Go type: time
	    startDate: any;
	    // Go type: time
	    endDate?: any;
	
	    static createFrom(source: any = {}) {
	        return new Budget(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.categoryId = source["categoryId"];
	        this.amountCents = source["amountCents"];
	        this.period = source["period"];
	        this.rollsOver = source["rollsOver"];
	        this.startDate = this.convertValues(source["startDate"], null);
	        this.endDate = this.convertValues(source["endDate"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Category {
	    id: number;
	    name: string;
	    parentId?: number;
	    color: string;
	    icon: string;
	    isIncome: boolean;
	    isSystem: boolean;
	    sortOrder: number;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.parentId = source["parentId"];
	        this.color = source["color"];
	        this.icon = source["icon"];
	        this.isIncome = source["isIncome"];
	        this.isSystem = source["isSystem"];
	        this.sortOrder = source["sortOrder"];
	    }
	}
	export class Transaction {
	    id: number;
	    accountId: number;
	    // Go type: time
	    date: any;
	    amountCents: number;
	    payee: string;
	    payeeNormalized: string;
	    notes: string;
	    categoryId?: number;
	    isTransfer: boolean;
	    transferPairId?: number;
	    isReconciled: boolean;
	    importHash: string;
	    importSource: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Transaction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.accountId = source["accountId"];
	        this.date = this.convertValues(source["date"], null);
	        this.amountCents = source["amountCents"];
	        this.payee = source["payee"];
	        this.payeeNormalized = source["payeeNormalized"];
	        this.notes = source["notes"];
	        this.categoryId = source["categoryId"];
	        this.isTransfer = source["isTransfer"];
	        this.transferPairId = source["transferPairId"];
	        this.isReconciled = source["isReconciled"];
	        this.importHash = source["importHash"];
	        this.importSource = source["importSource"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

