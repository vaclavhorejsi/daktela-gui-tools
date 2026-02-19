#import <Cocoa/Cocoa.h>
#include <stdlib.h>

extern void goTrayCallback(int tag);

@interface TrayTarget : NSObject
- (void)itemClicked:(NSMenuItem*)item;
@end

@implementation TrayTarget
- (void)itemClicked:(NSMenuItem*)item {
	goTrayCallback((int)[item tag]);
}
@end

static TrayTarget* _target;
static NSStatusItem* _statusItem;
static NSMutableArray<NSMenuItem*>* _historyItems;

void trayInit(const unsigned char* png, int pngLen) {
	dispatch_async(dispatch_get_main_queue(), ^{
		_target = [TrayTarget new];
		_historyItems = [NSMutableArray array];

		_statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];

		NSData* d = [NSData dataWithBytes:png length:pngLen];
		NSImage* img = [[NSImage alloc] initWithData:d];
		img.size = NSMakeSize(18, 18);
		img.template = YES;
		_statusItem.button.image = img;
		_statusItem.button.toolTip = @"Mountly";

		NSMenu* m = [NSMenu new];

		void (^add)(NSString*, int) = ^(NSString* title, int tag) {
			NSMenuItem* it = [[NSMenuItem alloc] initWithTitle:title action:@selector(itemClicked:) keyEquivalent:@""];
			it.target = _target;
			it.tag = tag;
			[m addItem:it];
		};

		add(@"Connect SSH  (Ctrl+Shift+S)", 1);
		add(@"Mount  (Ctrl+Shift+D)", 2);
		[m addItem:[NSMenuItem separatorItem]];

		for (int i = 0; i < 10; i++) {
			NSMenuItem* parent = [[NSMenuItem alloc] initWithTitle:@"" action:nil keyEquivalent:@""];
			parent.hidden = YES;

			NSMenu* sub = [NSMenu new];

			NSMenuItem* mountItem = [[NSMenuItem alloc] initWithTitle:@"Mount" action:@selector(itemClicked:) keyEquivalent:@""];
			mountItem.target = _target;
			mountItem.tag = 100 + i;
			[sub addItem:mountItem];

			NSMenuItem* connItem = [[NSMenuItem alloc] initWithTitle:@"Connect SSH" action:@selector(itemClicked:) keyEquivalent:@""];
			connItem.target = _target;
			connItem.tag = 200 + i;
			[sub addItem:connItem];

			parent.submenu = sub;
			[m addItem:parent];
			[_historyItems addObject:parent];
		}

		[m addItem:[NSMenuItem separatorItem]];
		add(@"Exit", 99);

		_statusItem.menu = m;
	});
}

void trayUpdateHistory(const char** titles, int count) {
	NSMutableArray* arr = [NSMutableArray array];
	for (int i = 0; i < count; i++) {
		[arr addObject:[NSString stringWithUTF8String:titles[i]]];
	}
	dispatch_async(dispatch_get_main_queue(), ^{
		for (NSUInteger i = 0; i < [_historyItems count]; i++) {
			NSMenuItem* p = _historyItems[i];
			if ((int)i < count) {
				p.title = arr[i];
				p.hidden = NO;
			} else {
				p.hidden = YES;
			}
		}
	});
}
